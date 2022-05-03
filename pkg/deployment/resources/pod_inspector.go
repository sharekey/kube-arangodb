//
// DISCLAIMER
//
// Copyright 2016-2022 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/kube-arangodb/pkg/deployment/patch"

	"github.com/arangodb/kube-arangodb/pkg/util/errors"
	inspectorInterface "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector"

	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"

	api "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1"
	"github.com/arangodb/kube-arangodb/pkg/metrics"
	"github.com/arangodb/kube-arangodb/pkg/util"
	"github.com/arangodb/kube-arangodb/pkg/util/k8sutil"
	podv1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/pod/v1"
)

var (
	inspectedPodsCounters     = metrics.MustRegisterCounterVec(metricsComponent, "inspected_pods", "Number of pod inspections per deployment", metrics.DeploymentName)
	inspectPodsDurationGauges = metrics.MustRegisterGaugeVec(metricsComponent, "inspect_pods_duration", "Amount of time taken by a single inspection of all pods for a deployment (in sec)", metrics.DeploymentName)
)

const (
	podScheduleTimeout              = time.Minute                // How long we allow the schedule to take scheduling a pod.
	recheckSoonPodInspectorInterval = util.Interval(time.Second) // Time between Pod inspection if we think something will change soon
	maxPodInspectorInterval         = util.Interval(time.Hour)   // Maximum time between Pod inspection (if nothing else happens)
)

// InspectPods lists all pods that belong to the given deployment and updates
// the member status of the deployment accordingly.
// Returns: Interval_till_next_inspection, error
func (r *Resources) InspectPods(ctx context.Context, cachedStatus inspectorInterface.Inspector) (util.Interval, error) {
	log := r.log
	start := time.Now()
	apiObject := r.context.GetAPIObject()
	deploymentName := apiObject.GetName()
	var events []*k8sutil.Event
	nextInterval := maxPodInspectorInterval // Large by default, will be made smaller if needed in the rest of the function
	defer metrics.SetDuration(inspectPodsDurationGauges.WithLabelValues(deploymentName), start)

	status, lastVersion := r.context.GetStatus()
	var podNamesWithScheduleTimeout []string
	var unscheduledPodNames []string

	err := cachedStatus.Pod().V1().Iterate(func(pod *v1.Pod) error {
		if k8sutil.IsArangoDBImageIDAndVersionPod(pod) {
			// Image ID pods are not relevant to inspect here
			return nil
		}

		// Pod belongs to this deployment, update metric
		inspectedPodsCounters.WithLabelValues(deploymentName).Inc()

		memberStatus, group, found := status.Members.MemberStatusByPodName(pod.GetName())
		if !found {
			log.Warn().Str("pod", pod.GetName()).Strs("existing-pods", status.Members.PodNames()).Msg("no memberstatus found for pod")
			if k8sutil.IsPodMarkedForDeletion(pod) && len(pod.GetFinalizers()) > 0 {
				// Strange, pod belongs to us, but we have no member for it.
				// Remove all finalizers, so it can be removed.
				log.Warn().Msg("Pod belongs to this deployment, but we don't know the member. Removing all finalizers")
				_, err := k8sutil.RemovePodFinalizers(ctx, r.context.GetCachedStatus(), log, r.context.PodsModInterface(), pod, pod.GetFinalizers(), false)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to update pod (to remove all finalizers)")
					return errors.WithStack(err)
				}
			}
			return nil
		}

		spec := r.context.GetSpec()
		coreContainers := spec.GetCoreContainers(group)

		// Update state
		updateMemberStatusNeeded := false
		if k8sutil.IsPodSucceeded(pod, coreContainers) {
			// Pod has terminated with exit code 0.
			wasTerminated := memberStatus.Conditions.IsTrue(api.ConditionTypeTerminated)
			if memberStatus.Conditions.Update(api.ConditionTypeTerminated, true, "Pod Succeeded", "") {
				log.Debug().Str("pod-name", pod.GetName()).Msg("Updating member condition Terminated to true: Pod Succeeded")
				updateMemberStatusNeeded = true
				nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
				if !wasTerminated {
					// Record termination time
					now := meta.Now()
					memberStatus.RecentTerminations = append(memberStatus.RecentTerminations, now)
				}
			}
		} else if k8sutil.IsPodFailed(pod, coreContainers) {
			// Pod has terminated with at least 1 container with a non-zero exit code.
			wasTerminated := memberStatus.Conditions.IsTrue(api.ConditionTypeTerminated)
			if memberStatus.Conditions.Update(api.ConditionTypeTerminated, true, "Pod Failed", "") {
				if containers := k8sutil.GetFailedContainerNames(pod.Status.InitContainerStatuses); len(containers) > 0 {
					for _, container := range containers {
						switch container {
						case api.ServerGroupReservedInitContainerNameVersionCheck:
							if c, ok := k8sutil.GetAnyContainerStatusByName(pod.Status.InitContainerStatuses, container); ok {
								if t := c.State.Terminated; t != nil && t.ExitCode == 11 {
									memberStatus.Upgrade = true
									updateMemberStatusNeeded = true
								}
							}
						case api.ServerGroupReservedInitContainerNameUpgrade:
							memberStatus.Conditions.Update(api.ConditionTypeUpgradeFailed, true, "Upgrade Failed", "")
						}

						if c, ok := k8sutil.GetAnyContainerStatusByName(pod.Status.InitContainerStatuses, container); ok {
							if t := c.State.Terminated; t != nil && t.ExitCode != 0 {
								log.Warn().Str("member", memberStatus.ID).
									Str("pod", pod.GetName()).
									Str("container", container).
									Str("uid", string(pod.GetUID())).
									Int32("exit-code", t.ExitCode).
									Str("reason", t.Reason).
									Str("message", t.Message).
									Int32("signal", t.Signal).
									Time("started", t.StartedAt.Time).
									Time("finished", t.FinishedAt.Time).
									Msgf("Pod failed in unexpected way: Init Container failed")
							}
						}
					}
				}

				if containers := k8sutil.GetFailedContainerNames(pod.Status.ContainerStatuses); len(containers) > 0 {
					for _, container := range containers {
						if c, ok := k8sutil.GetAnyContainerStatusByName(pod.Status.ContainerStatuses, container); ok {
							if t := c.State.Terminated; t != nil && t.ExitCode != 0 {
								log.Warn().Str("member", memberStatus.ID).
									Str("pod", pod.GetName()).
									Str("container", container).
									Str("uid", string(pod.GetUID())).
									Int32("exit-code", t.ExitCode).
									Str("reason", t.Reason).
									Str("message", t.Message).
									Int32("signal", t.Signal).
									Time("started", t.StartedAt.Time).
									Time("finished", t.FinishedAt.Time).
									Msgf("Pod failed in unexpected way: Core Container failed")
							}
						}
					}
				}

				log.Debug().Str("pod-name", pod.GetName()).Msg("Updating member condition Terminated to true: Pod Failed")
				updateMemberStatusNeeded = true
				nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
				if !wasTerminated {
					// Record termination time
					now := meta.Now()
					memberStatus.RecentTerminations = append(memberStatus.RecentTerminations, now)
				}
			}
		}

		if k8sutil.IsPodScheduled(pod) {
			if _, ok := pod.Labels[k8sutil.LabelKeyArangoScheduled]; !ok {
				// Adding scheduled label to the pod
				l := addLabel(pod.Labels, k8sutil.LabelKeyArangoScheduled, "1")

				if err := r.context.ApplyPatchOnPod(ctx, pod, patch.ItemReplace(patch.NewPath("metadata", "labels"), l)); err != nil {
					log.Error().Err(err).Msgf("Unable to update scheduled labels")
				}
			}
		}

		// Topology labels
		tv, tok := pod.Labels[k8sutil.LabelKeyArangoTopology]
		zv, zok := pod.Labels[k8sutil.LabelKeyArangoZone]

		if t, ts := status.Topology, memberStatus.Topology; t.Enabled() && t.IsTopologyOwned(ts) {
			if tid, tz := string(t.ID), fmt.Sprintf("%d", ts.Zone); !tok || !zok || tv != tid || zv != tz {
				l := addLabel(pod.Labels, k8sutil.LabelKeyArangoTopology, tid)
				l = addLabel(l, k8sutil.LabelKeyArangoZone, tz)

				if err := r.context.ApplyPatchOnPod(ctx, pod, patch.ItemReplace(patch.NewPath("metadata", "labels"), l)); err != nil {
					log.Error().Err(err).Msgf("Unable to update topology labels")
				}
			}
		} else {
			if tok || zok {
				l := removeLabel(pod.Labels, k8sutil.LabelKeyArangoTopology)
				l = removeLabel(l, k8sutil.LabelKeyArangoZone)

				if err := r.context.ApplyPatchOnPod(ctx, pod, patch.ItemReplace(patch.NewPath("metadata", "labels"), l)); err != nil {
					log.Error().Err(err).Msgf("Unable to remove topology labels")
				}
			}
		}
		// End of Topology labels

		// Reachable state
		if state, ok := r.context.GetMembersState().MemberState(memberStatus.ID); ok {
			if state.IsReachable() {
				if memberStatus.Conditions.Update(api.ConditionTypeReachable, true, "ArangoDB is reachable", "") {
					updateMemberStatusNeeded = true
					nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
				}
			} else {
				if memberStatus.Conditions.Update(api.ConditionTypeReachable, false, "ArangoDB is not reachable", "") {
					updateMemberStatusNeeded = true
					nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
				}
			}
		}

		if k8sutil.IsPodReady(pod) && k8sutil.AreContainersReady(pod, coreContainers) {
			// Pod is now ready
			if anyOf(memberStatus.Conditions.Update(api.ConditionTypeReady, true, "Pod Ready", ""),
				memberStatus.Conditions.Update(api.ConditionTypeStarted, true, "Pod Started", ""),
				memberStatus.Conditions.Update(api.ConditionTypeServing, true, "Pod Serving", "")) {
				log.Debug().Str("pod-name", pod.GetName()).Msg("Updating member condition Ready, Started & Serving to true")

				if status.Topology.IsTopologyOwned(memberStatus.Topology) {
					nodes, err := cachedStatus.Node().V1()
					if err == nil {
						node, ok := nodes.GetSimple(pod.Spec.NodeName)
						if ok {
							label, ok := node.Labels[status.Topology.Label]
							if ok {
								memberStatus.Topology.Label = label
							}
						}
					}
				}

				memberStatus.IsInitialized = true // Require future pods for this member to have an existing UUID (in case of dbserver).
				updateMemberStatusNeeded = true
				nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
			}
		} else if k8sutil.AreContainersReady(pod, coreContainers) {
			// Pod is not ready, but core containers are fine
			if anyOf(memberStatus.Conditions.Update(api.ConditionTypeReady, false, "Pod Not Ready", ""),
				memberStatus.Conditions.Update(api.ConditionTypeServing, true, "Pod is still serving", "")) {
				log.Debug().Str("pod-name", pod.GetName()).Msg("Updating member condition Ready to false, while all core containers are ready")
				updateMemberStatusNeeded = true
				nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
			}
		} else {
			// Pod is not ready
			if anyOf(memberStatus.Conditions.Update(api.ConditionTypeReady, false, "Pod Not Ready", ""),
				memberStatus.Conditions.Update(api.ConditionTypeServing, false, "Pod Core containers are not ready", strings.Join(coreContainers, ", "))) {
				log.Debug().Str("pod-name", pod.GetName()).Msg("Updating member condition Ready & Serving to false")
				updateMemberStatusNeeded = true
				nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
			}
		}

		if k8sutil.IsPodNotScheduledFor(pod, podScheduleTimeout) {
			// Pod cannot be scheduled for to long
			log.Debug().Str("pod-name", pod.GetName()).Msg("Pod scheduling timeout")
			podNamesWithScheduleTimeout = append(podNamesWithScheduleTimeout, pod.GetName())
		} else if !k8sutil.IsPodScheduled(pod) {
			unscheduledPodNames = append(unscheduledPodNames, pod.GetName())
		}

		if k8sutil.IsPodMarkedForDeletion(pod) {
			if memberStatus.Conditions.Update(api.ConditionTypeTerminating, true, "Pod marked for deletion", "") {
				updateMemberStatusNeeded = true
				log.Debug().Str("pod-name", pod.GetName()).Msg("Pod marked as terminating")
			}
			// Process finalizers
			if x, err := r.runPodFinalizers(ctx, pod, memberStatus, func(m api.MemberStatus) error {
				updateMemberStatusNeeded = true
				memberStatus = m
				return nil
			}); err != nil {
				// Only log here, since we'll be called to try again.
				log.Warn().Err(err).Msg("Failed to run pod finalizers")
			} else {
				nextInterval = nextInterval.ReduceTo(x)
			}
		}

		if updateMemberStatusNeeded {
			if err := status.Members.Update(memberStatus, group); err != nil {
				return errors.WithStack(err)
			}
		}

		return nil
	}, podv1.FilterPodsByLabels(k8sutil.LabelsForDeployment(deploymentName, "")))
	if err != nil {
		return 0, err
	}

	// Go over all members, check for missing pods
	status.Members.ForeachServerGroup(func(group api.ServerGroup, members api.MemberStatusList) error {
		for _, m := range members {
			if podName := m.PodName; podName != "" {
				if _, exists := cachedStatus.Pod().V1().GetSimple(podName); !exists {
					log.Debug().Str("pod-name", podName).Msg("Does not exist")
					switch m.Phase {
					case api.MemberPhaseNone, api.MemberPhasePending:
						// Do nothing
						log.Debug().Str("pod-name", podName).Msg("PodPhase is None, waiting for the pod to be recreated")
					case api.MemberPhaseShuttingDown, api.MemberPhaseUpgrading, api.MemberPhaseFailed, api.MemberPhaseRotateStart, api.MemberPhaseRotating:
						// Shutdown was intended, so not need to do anything here.
						// Just mark terminated
						wasTerminated := m.Conditions.IsTrue(api.ConditionTypeTerminated)
						if m.Conditions.Update(api.ConditionTypeTerminated, true, "Pod Terminated", "") {
							if !wasTerminated {
								// Record termination time
								now := meta.Now()
								m.RecentTerminations = append(m.RecentTerminations, now)
							}
							// Save it
							if err := status.Members.Update(m, group); err != nil {
								return errors.WithStack(err)
							}
						}
					default:
						log.Debug().Str("pod-name", podName).Msg("Pod is gone")
						m.Phase = api.MemberPhaseNone // This is trigger a recreate of the pod.
						// Create event
						nextInterval = nextInterval.ReduceTo(recheckSoonPodInspectorInterval)
						events = append(events, k8sutil.NewPodGoneEvent(podName, group.AsRole(), apiObject))
						updateMemberNeeded := false
						if m.Conditions.Update(api.ConditionTypeReady, false, "Pod Does Not Exist", "") {
							updateMemberNeeded = true
						}
						wasTerminated := m.Conditions.IsTrue(api.ConditionTypeTerminated)
						if m.Conditions.Update(api.ConditionTypeTerminated, true, "Pod Does Not Exist", "") {
							if !wasTerminated {
								// Record termination time
								now := meta.Now()
								m.RecentTerminations = append(m.RecentTerminations, now)
							}
							updateMemberNeeded = true
						}
						if updateMemberNeeded {
							// Save it
							if err := status.Members.Update(m, group); err != nil {
								return errors.WithStack(err)
							}
						}
					}
				}
			}
		}
		return nil
	})

	spec := r.context.GetSpec()
	allMembersReady := status.Members.AllMembersReady(spec.GetMode(), spec.Sync.IsEnabled())
	status.Conditions.Update(api.ConditionTypeReady, allMembersReady, "", "")

	// Update conditions
	if len(podNamesWithScheduleTimeout) > 0 {
		if status.Conditions.Update(api.ConditionTypePodSchedulingFailure, true,
			"Pods Scheduling Timeout",
			fmt.Sprintf("The following pods cannot be scheduled: %v", podNamesWithScheduleTimeout)) {
			r.context.CreateEvent(k8sutil.NewPodsSchedulingFailureEvent(podNamesWithScheduleTimeout, r.context.GetAPIObject()))
		}
	} else if status.Conditions.IsTrue(api.ConditionTypePodSchedulingFailure) &&
		len(unscheduledPodNames) == 0 {
		if status.Conditions.Update(api.ConditionTypePodSchedulingFailure, false,
			"Pods Scheduling Resolved",
			"No pod reports a scheduling timeout") {
			r.context.CreateEvent(k8sutil.NewPodsSchedulingResolvedEvent(r.context.GetAPIObject()))
		}
	}

	// Save status
	if err := r.context.UpdateStatus(ctx, status, lastVersion); err != nil {
		return 0, errors.WithStack(err)
	}

	// Create events
	for _, evt := range events {
		r.context.CreateEvent(evt)
	}
	return nextInterval, nil
}

func addLabel(labels map[string]string, key, value string) map[string]string {
	if labels != nil {
		labels[key] = value
		return labels
	}

	return map[string]string{
		key: value,
	}
}

func removeLabel(labels map[string]string, key string) map[string]string {
	if labels == nil {
		return map[string]string{}
	}

	delete(labels, key)

	return labels
}

func anyOf(bools ...bool) bool {
	for _, b := range bools {
		if b {
			return true
		}
	}

	return false
}
