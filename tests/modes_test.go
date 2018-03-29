//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Jan Christoph Uhde <jan@uhdejc.com>
//
package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dchest/uniuri"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1alpha"
	kubeArangoClient "github.com/arangodb/kube-arangodb/pkg/client"
	"github.com/arangodb/kube-arangodb/pkg/util"
)

// test if deployment comes up in production mode if there are enough nodes and fails if there are too few
func TestProduction(t *testing.T) {
	longOrSkip(t)

	mode := api.DeploymentModeCluster
	engine := api.StorageEngineRocksDB

	k8sNameSpace := getNamespace(t)
	k8sClient := mustNewKubeClient(t)
	deploymentClient := kubeArangoClient.MustNewInCluster()
	deploymentTemplate := newDeployment(strings.Replace(fmt.Sprintf("tprod-%s-%s-%s", mode[:2], engine[:2], uniuri.NewLen(4)), ".", "", -1))
	deploymentTemplate.Spec.Mode = api.NewMode(mode)
	deploymentTemplate.Spec.StorageEngine = api.NewStorageEngine(engine)
	deploymentTemplate.Spec.TLS = api.TLSSpec{} // should auto-generate cert
	deploymentTemplate.Spec.Environment = api.NewEnvironment(api.EnvironmentProduction)
	deploymentTemplate.Spec.DBServers.Count = util.NewInt(4)
	deploymentTemplate.Spec.SetDefaults(deploymentTemplate.GetName()) // this must be last

	dbserverCount := *deploymentTemplate.Spec.DBServers.Count
	if dbserverCount < 3 {
		t.Fatalf("Not enough DBServers to run this test: server count %d", dbserverCount)
	}

	options := meta_v1.ListOptions{}
	nodeList, err := k8sClient.CoreV1().Nodes().List(options)
	if err != nil {
		t.Fatalf("Unable to receive node list: %v", err)
	}

	numNodes := len(nodeList.Items)
	failExpected := false

	if numNodes < dbserverCount {
		failExpected = true
	}

	// Create deployment
	_, err = deploymentClient.DatabaseV1alpha().ArangoDeployments(k8sNameSpace).Create(deploymentTemplate)
	if err != nil {
		t.Fatalf("Create deployment failed: %v", err)
	}

	deployment, err := waitUntilDeployment(deploymentClient, deploymentTemplate.GetName(), k8sNameSpace, deploymentIsReady())
	if failExpected {
		if err == nil {
			t.Fatalf("Deployment is up and running when it should not! There are not enough nodes(%d) for all DBServers(%d) in production modes.", numNodes, dbserverCount)
		}
	} else {
		if err != nil {
			t.Fatalf("Deployment not running in time: %v", err)
		}
		// Create a database client
		ctx := context.Background()
		DBClient := mustNewArangodDatabaseClient(ctx, k8sClient, deployment, t)

		if err := waitUntilArangoDeploymentHealthy(deployment, DBClient, k8sClient, ""); err != nil {
			t.Fatalf("Deployment not healthy in time: %v", err)
		}
	}

	// Cleanup
	removeDeployment(deploymentClient, deploymentTemplate.GetName(), k8sNameSpace)
}
