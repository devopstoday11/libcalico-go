// Copyright (c) 2017 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientv3_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/testutils"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

var _ = testutils.E2eDatastoreDescribe("BGPPeer tests", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()
	name1 := "bgppeer-1"
	name2 := "bgppeer-2"
	spec1 := apiv3.BGPPeerSpec{
		Node:     "node1",
		PeerIP:   "10.0.0.1",
		ASNumber: numorstring.ASNumber(6512),
	}
	spec2 := apiv3.BGPPeerSpec{
		Node:     "node2",
		PeerIP:   "20.0.0.1",
		ASNumber: numorstring.ASNumber(6511),
	}

	DescribeTable("BGPPeer e2e CRUD tests",
		func(name1, name2 string, spec1, spec2 apiv3.BGPPeerSpec) {
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			By("Updating the BGPPeer before it is created")
			_, outError := c.BGPPeers().Update(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", CreationTimestamp: metav1.Now(), UID: "test-fail-bgppeer"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: BGPPeer(" + name1 + ") with error:"))

			By("Attempting to creating a new BGPPeer with name1/spec1 and a non-empty ResourceVersion")
			_, outError = c.BGPPeers().Create(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "12345"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("error with field Metadata.ResourceVersion = '12345' (field must not be set for a Create request)"))

			By("Creating a new BGPPeer with name1/spec1")
			res1, outError := c.BGPPeers().Create(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name1},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res1, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec1)

			// Track the version of the original data for name1.
			rv1_1 := res1.ResourceVersion

			By("Attempting to create the same BGPPeer with name1 but with spec2")
			_, outError = c.BGPPeers().Create(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name1},
				Spec:       spec2,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("resource already exists: BGPPeer(" + name1 + ")"))

			By("Getting BGPPeer (name1) and comparing the output against spec1")
			res, outError := c.BGPPeers().Get(ctx, name1, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec1)
			Expect(res.ResourceVersion).To(Equal(res1.ResourceVersion))

			By("Getting BGPPeer (name2) before it is created")
			_, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: BGPPeer(" + name2 + ") with error:"))

			By("Listing all the BGPPeers, expecting a single result with name1/spec1")
			outList, outError := c.BGPPeers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(1))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec1)

			By("Creating a new BGPPeer with name2/spec2")
			res2, outError := c.BGPPeers().Create(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name2},
				Spec:       spec2,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res2, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name2, spec2)

			By("Getting BGPPeer (name2) and comparing the output against spec2")
			res, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res2, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name2, spec2)
			Expect(res.ResourceVersion).To(Equal(res2.ResourceVersion))

			By("Listing all the BGPPeers, expecting a two results with name1/spec1 and name2/spec2")
			outList, outError = c.BGPPeers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(2))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec1)
			testutils.ExpectResource(&outList.Items[1], apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name2, spec2)

			By("Updating BGPPeer name1 with spec2")
			res1.Spec = spec2
			res1, outError = c.BGPPeers().Update(ctx, res1, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res1, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec2)

			By("Attempting to update the BGPPeer without a Creation Timestamp")
			res, outError = c.BGPPeers().Update(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", UID: "test-fail-bgppeer"},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(res).To(BeNil())
			Expect(outError.Error()).To(Equal("error with field Metadata.CreationTimestamp = '0001-01-01 00:00:00 +0000 UTC' (field must be set for an Update request)"))

			By("Attempting to update the BGPPeer without a UID")
			res, outError = c.BGPPeers().Update(ctx, &apiv3.BGPPeer{
				ObjectMeta: metav1.ObjectMeta{Name: name1, ResourceVersion: "1234", CreationTimestamp: metav1.Now()},
				Spec:       spec1,
			}, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(res).To(BeNil())
			Expect(outError.Error()).To(Equal("error with field Metadata.UID = '' (field must be set for an Update request)"))

			// Track the version of the updated name1 data.
			rv1_2 := res1.ResourceVersion

			By("Updating BGPPeer name1 without specifying a resource version")
			res1.Spec = spec1
			res1.ObjectMeta.ResourceVersion = ""
			_, outError = c.BGPPeers().Update(ctx, res1, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("error with field Metadata.ResourceVersion = '' (field must be set for an Update request)"))

			By("Updating BGPPeer name1 using the previous resource version")
			res1.Spec = spec1
			res1.ResourceVersion = rv1_1
			_, outError = c.BGPPeers().Update(ctx, res1, options.SetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(Equal("update conflict: BGPPeer(" + name1 + ")"))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Getting BGPPeer (name1) with the original resource version and comparing the output against spec1")
				res, outError = c.BGPPeers().Get(ctx, name1, options.GetOptions{ResourceVersion: rv1_1})
				Expect(outError).NotTo(HaveOccurred())
				testutils.ExpectResource(res, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec1)
				Expect(res.ResourceVersion).To(Equal(rv1_1))
			}

			By("Getting BGPPeer (name1) with the updated resource version and comparing the output against spec2")
			res, outError = c.BGPPeers().Get(ctx, name1, options.GetOptions{ResourceVersion: rv1_2})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(res, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec2)
			Expect(res.ResourceVersion).To(Equal(rv1_2))

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Listing BGPPeers with the original resource version and checking for a single result with name1/spec1")
				outList, outError = c.BGPPeers().List(ctx, options.ListOptions{ResourceVersion: rv1_1})
				Expect(outError).NotTo(HaveOccurred())
				Expect(outList.Items).To(HaveLen(1))
				testutils.ExpectResource(&outList.Items[0], apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec1)
			}

			By("Listing BGPPeers with the latest resource version and checking for two results with name1/spec2 and name2/spec2")
			outList, outError = c.BGPPeers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(2))
			testutils.ExpectResource(&outList.Items[0], apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec2)
			testutils.ExpectResource(&outList.Items[1], apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name2, spec2)

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Deleting BGPPeer (name1) with the old resource version")
				_, outError = c.BGPPeers().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_1})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(Equal("update conflict: BGPPeer(" + name1 + ")"))
			}

			By("Deleting BGPPeer (name1) with the new resource version")
			dres, outError := c.BGPPeers().Delete(ctx, name1, options.DeleteOptions{ResourceVersion: rv1_2})
			Expect(outError).NotTo(HaveOccurred())
			testutils.ExpectResource(dres, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name1, spec2)

			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Updating BGPPeer name2 with a 2s TTL and waiting for the entry to be deleted")
				_, outError = c.BGPPeers().Update(ctx, res2, options.SetOptions{TTL: 2 * time.Second})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)
				_, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
				_, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(ContainSubstring("resource does not exist: BGPPeer(" + name2 + ") with error:"))

				By("Creating BGPPeer name2 with a 2s TTL and waiting for the entry to be deleted")
				_, outError = c.BGPPeers().Create(ctx, &apiv3.BGPPeer{
					ObjectMeta: metav1.ObjectMeta{Name: name2},
					Spec:       spec2,
				}, options.SetOptions{TTL: 2 * time.Second})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)
				_, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
				_, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
				Expect(outError).To(HaveOccurred())
				Expect(outError.Error()).To(ContainSubstring("resource does not exist: BGPPeer(" + name2 + ") with error:"))
			}

			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				By("Attempting to deleting BGPPeer (name2)")
				dres, outError = c.BGPPeers().Delete(ctx, name2, options.DeleteOptions{})
				Expect(outError).NotTo(HaveOccurred())
				testutils.ExpectResource(dres, apiv3.KindBGPPeer, testutils.ExpectNoNamespace, name2, spec2)
			}

			By("Attempting to deleting BGPPeer (name2) again")
			_, outError = c.BGPPeers().Delete(ctx, name2, options.DeleteOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: BGPPeer(" + name2 + ") with error:"))

			By("Listing all BGPPeers and expecting no items")
			outList, outError = c.BGPPeers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(0))

			By("Getting BGPPeer (name2) and expecting an error")
			_, outError = c.BGPPeers().Get(ctx, name2, options.GetOptions{})
			Expect(outError).To(HaveOccurred())
			Expect(outError.Error()).To(ContainSubstring("resource does not exist: BGPPeer(" + name2 + ") with error:"))
		},

		// Test 1: Pass two fully populated BGPPeerSpecs and expect the series of operations to succeed.
		Entry("Two fully populated BGPPeerSpecs", name1, name2, spec1, spec2),
	)

	Describe("BGPPeer watch functionality", func() {
		It("should handle watch events for different resource versions and event types", func() {
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			By("Listing BGPPeers with the latest resource version and checking for two results with name1/spec2 and name2/spec2")
			outList, outError := c.BGPPeers().List(ctx, options.ListOptions{})
			Expect(outError).NotTo(HaveOccurred())
			Expect(outList.Items).To(HaveLen(0))
			rev0 := outList.ResourceVersion

			By("Configuring a BGPPeer name1/spec1 and storing the response")
			outRes1, err := c.BGPPeers().Create(
				ctx,
				&apiv3.BGPPeer{
					ObjectMeta: metav1.ObjectMeta{Name: name1},
					Spec:       spec1,
				},
				options.SetOptions{},
			)
			rev1 := outRes1.ResourceVersion

			By("Configuring a BGPPeer name2/spec2 and storing the response")
			outRes2, err := c.BGPPeers().Create(
				ctx,
				&apiv3.BGPPeer{
					ObjectMeta: metav1.ObjectMeta{Name: name2},
					Spec:       spec2,
				},
				options.SetOptions{},
			)

			By("Starting a watcher from revision rev1 - this should skip the first creation")
			w, err := c.BGPPeers().Watch(ctx, options.ListOptions{ResourceVersion: rev1})
			Expect(err).NotTo(HaveOccurred())
			testWatcher1 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher1.Stop()

			By("Deleting res1")
			_, err = c.BGPPeers().Delete(ctx, name1, options.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Checking for two events, create res2 and delete re1")
			testWatcher1.ExpectEvents(apiv3.KindBGPPeer, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes2,
				},
				{
					Type:     watch.Deleted,
					Previous: outRes1,
				},
			})
			testWatcher1.Stop()

			By("Starting a watcher from rev0 - this should get all events")
			w, err = c.BGPPeers().Watch(ctx, options.ListOptions{ResourceVersion: rev0})
			Expect(err).NotTo(HaveOccurred())
			testWatcher2 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher2.Stop()

			By("Modifying res2")
			outRes3, err := c.BGPPeers().Update(
				ctx,
				&apiv3.BGPPeer{
					ObjectMeta: outRes2.ObjectMeta,
					Spec:       spec1,
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			testWatcher2.ExpectEvents(apiv3.KindBGPPeer, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes1,
				},
				{
					Type:   watch.Added,
					Object: outRes2,
				},
				{
					Type:     watch.Deleted,
					Previous: outRes1,
				},
				{
					Type:     watch.Modified,
					Previous: outRes2,
					Object:   outRes3,
				},
			})
			testWatcher2.Stop()

			// Only etcdv3 supports watching a specific instance of a resource.
			if config.Spec.DatastoreType == apiconfig.EtcdV3 {
				By("Starting a watcher from rev0 watching name1 - this should get all events for name1")
				w, err = c.BGPPeers().Watch(ctx, options.ListOptions{Name: name1, ResourceVersion: rev0})
				Expect(err).NotTo(HaveOccurred())
				testWatcher2_1 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
				defer testWatcher2_1.Stop()
				testWatcher2_1.ExpectEvents(apiv3.KindBGPPeer, []watch.Event{
					{
						Type:   watch.Added,
						Object: outRes1,
					},
					{
						Type:     watch.Deleted,
						Previous: outRes1,
					},
				})
				testWatcher2_1.Stop()
			}

			By("Starting a watcher not specifying a rev - expect the current snapshot")
			w, err = c.BGPPeers().Watch(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			testWatcher3 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher3.Stop()
			testWatcher3.ExpectEvents(apiv3.KindBGPPeer, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes3,
				},
			})
			testWatcher3.Stop()

			By("Configuring BGPPeer name1/spec1 again and storing the response")
			outRes1, err = c.BGPPeers().Create(
				ctx,
				&apiv3.BGPPeer{
					ObjectMeta: metav1.ObjectMeta{Name: name1},
					Spec:       spec1,
				},
				options.SetOptions{},
			)

			By("Starting a watcher not specifying a rev - expect the current snapshot")
			w, err = c.BGPPeers().Watch(ctx, options.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			testWatcher4 := testutils.NewTestResourceWatch(config.Spec.DatastoreType, w)
			defer testWatcher4.Stop()
			testWatcher4.ExpectEventsAnyOrder(apiv3.KindBGPPeer, []watch.Event{
				{
					Type:   watch.Added,
					Object: outRes1,
				},
				{
					Type:   watch.Added,
					Object: outRes3,
				},
			})

			By("Cleaning the datastore and expecting deletion events for each configured resource (tests prefix deletes results in individual events for each key)")
			be.Clean()
			testWatcher4.ExpectEvents(apiv3.KindBGPPeer, []watch.Event{
				{
					Type:     watch.Deleted,
					Previous: outRes1,
				},
				{
					Type:     watch.Deleted,
					Previous: outRes3,
				},
			})
			testWatcher4.Stop()
		})
	})
})
