/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package notifications

import (
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNotifications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notifications Suite")
}

var _ = Describe("NotificationQueue", func() {
	var notification *types.Notification

	BeforeEach(func() {
		notification = &types.Notification{
			PlatformID: "123",
		}
	})

	Context("When queue is not full", func() {
		It("should add a notification", func() {
			notificationQueue := NewNotificationQueue(1)
			err := notificationQueue.Enqueue(notification)
			Expect(err).ToNot(HaveOccurred())
			Expect(notificationQueue.Next()).To(Equal(notification))
		})
	})

	Context("When queue is full", func() {
		It("enqueue should return error", func() {
			notificationQueue := NewNotificationQueue(0)
			err := notificationQueue.Enqueue(notification)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("notification queue is full"))
		})
	})

	Context("When queue is closed", func() {
		It("next should return error", func() {
			notificationQueue := NewNotificationQueue(0)
			notificationQueue.Close()
			_, err := notificationQueue.Next()
			Expect(err).To(Equal(ErrQueueClosed))
		})
	})

	Context("When queue is closed", func() {
		It("enqueue should return error", func() {
			notificationQueue := NewNotificationQueue(1)
			notificationQueue.Close()
			err := notificationQueue.Enqueue(nil)
			Expect(err).To(Equal(ErrQueueClosed))
		})
	})
})
