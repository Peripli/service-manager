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

package ws_test

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/ws"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWsConn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Websocket test suite")
}

func wsconnect(ctx *common.TestContext, path string) (*websocket.Conn, *http.Response, error) {
	smURL := ctx.Servers[common.SMServer].URL()
	smEndpoint, _ := url.Parse(smURL)
	return websocket.DefaultDialer.Dial("ws://"+smEndpoint.Host+path, nil)
}

var _ = Describe("WS", func() {
	var ctx *common.TestContext
	var wsconn *websocket.Conn
	var resp *http.Response
	var testWsHandler *wsHandler
	var testWsHandlers []*wsHandler
	var upgrader ws.Upgrader
	var work sync.WaitGroup

	wsPingTimeout := time.Second * 20

	JustBeforeEach(func() {
		testWsHandler = newWsHandler()
		testWsHandlers = append([]*wsHandler{testWsHandler}, testWsHandlers...)
		work = sync.WaitGroup{}
		work.Add(1)
		upgrader = ws.NewUpgrader(context.Background(), &work, &ws.UpgraderOptions{PingTimeout: wsPingTimeout})
		ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.RegisterControllers(&wsController{
				wsUpgrader: upgrader,
				wsHandlers: testWsHandlers,
			})
			return nil
		}).Build()

		var err error
		wsconn, resp, err = wsconnect(ctx, "/v1/testws")
		Expect(err).ShouldNot(HaveOccurred())
	})

	JustAfterEach(func() {
		if wsconn != nil {
			wsconn.Close()
		}
		testWsHandlers = nil
		ctx.Cleanup()
	})

	Describe("establish websocket connection", func() {
		Context("when response headers are set", func() {
			It("should receive response header", func() {
				Expect(resp.Header.Get("test-header")).To(Equal("test"))
			})
		})

		Context("when service manager sends data over ws connection", func() {
			It("client should receive messages", func() {
				message := "from server"
				testWsHandler.doSend <- message
				assertMessage(wsconn, message)
			})
		})

		Context("when client sends data over ws connection", func() {
			It("service manager should receive the message", func() {
				message := "from client"
				err := wsconn.WriteMessage(websocket.TextMessage, []byte(message))
				Expect(err).ShouldNot(HaveOccurred())

				var expectedMessage string
				Eventually(testWsHandler.receivedMessages).Should(Receive(&expectedMessage))
				Expect(expectedMessage).To(Equal(message))
			})
		})

		Context("when connection timeout is reached", func() {
			BeforeEach(func() {
				wsPingTimeout = time.Millisecond
			})

			It("client should receive close message", func() {
				var expectedError string
				Eventually(testWsHandler.receivedErrors).Should(Receive(&expectedError))
				Expect(expectedError).To(ContainSubstring("timeout"))
			})
		})

		Context("when ping is sent before connection timeout is reached", func() {
			var pongCh chan struct{}

			BeforeEach(func() {
				wsPingTimeout = time.Second * 2
			})

			JustBeforeEach(func() {
				pongCh = make(chan struct{})
				wsconn.SetReadDeadline(time.Time{})
				wsconn.SetPongHandler(func(data string) error {
					Expect(data).To(Equal("pingping"))
					close(pongCh)
					return nil
				})
				go func() {
					_, _, err := wsconn.ReadMessage()
					Expect(err).Should(HaveOccurred())
				}()
			})

			It("connection should receive pong", func(done Done) {
				err := wsconn.WriteMessage(websocket.PingMessage, []byte("pingping"))
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(pongCh, 5).ShouldNot(Receive())
				close(done)
			}, 3)

			It("connection timeout should be refreshed", func(done Done) {
				time.Sleep(time.Second)
				err := wsconn.WriteMessage(websocket.PingMessage, []byte("pingping"))
				Expect(err).ShouldNot(HaveOccurred())
				time.Sleep(time.Second + time.Millisecond*500)

				message := "from client"
				err = wsconn.WriteMessage(websocket.TextMessage, []byte(message))
				Expect(err).ShouldNot(HaveOccurred())

				var expectedMessage string
				Eventually(testWsHandler.receivedMessages).Should(Receive(&expectedMessage))
				Expect(expectedMessage).To(Equal(message))

				close(done)
			}, 4)
		})

		Context("when 2 websocket connections are opened", func() {
			var wsconn2 *websocket.Conn
			testWsHandler2 := newWsHandler()

			BeforeEach(func() {
				testWsHandlers = append(testWsHandlers, testWsHandler2)
			})

			JustBeforeEach(func() {
				var err error
				wsconn2, _, err = wsconnect(ctx, "/v1/testws")
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should be able to send data over both", func() {
				testWsHandler.doSend <- "msg1"
				testWsHandler2.doSend <- "msg2"
				assertMessage(wsconn, "msg1")
				assertMessage(wsconn2, "msg2")
			})

			It("should be able to send data over one of them even if the other is closed", func() {
				err := wsconn2.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
				Expect(err).ShouldNot(HaveOccurred())
				var receivedError string
				Eventually(testWsHandler2.receivedErrors).Should(Receive(&receivedError))
				Expect(receivedError).To(ContainSubstring("websocket: close"))
				testWsHandler.doSend <- "msg"
				assertMessage(wsconn, "msg")
			})
		})

		Context("when upgrader is shutdown", func() {
			It("should close all connections", func() {
				err := upgrader.Shutdown()
				work.Wait()
				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})
})

func assertMessage(c *websocket.Conn, expectedMsg string) {
	_, msg, err := c.ReadMessage()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(string(msg)).To(Equal(expectedMsg))
}

type wsController struct {
	wsUpgrader ws.Upgrader
	wsHandlers []*wsHandler
}

func (wsc *wsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   "/v1/testws",
			},
			Handler: wsc.handle,
		},
	}
}

func (wsc *wsController) handle(req *web.Request) (*web.Response, error) {
	rw := req.HijackResponseWriter()
	header := http.Header{
		"test-header": []string{"test"},
	}
	wsConn, err := wsc.wsUpgrader.Upgrade(rw, req.Request, header)
	if err != nil {
		return nil, err
	}

	if wsc.wsHandlers == nil {
		panic("should not panic")
	}

	handler := wsc.wsHandlers[0]
	if len(wsc.wsHandlers) > 1 {
		wsc.wsHandlers = wsc.wsHandlers[1:]
	} else {
		wsc.wsHandlers = nil
	}

	go handler.Write(wsConn)
	go handler.Read(wsConn)

	return &web.Response{}, nil
}

type wsHandler struct {
	doSend chan string

	receivedMessages chan string
	receivedErrors   chan string
}

func newWsHandler() *wsHandler {
	return &wsHandler{
		doSend:           make(chan string),
		receivedMessages: make(chan string),
		receivedErrors:   make(chan string),
	}
}

func (h *wsHandler) Write(c *ws.Conn) {
	for {
		select {
		case <-c.Shutdown:
			c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
			c.Close()
			return
		case msg := <-h.doSend:
			err := c.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				return
			}
		default:
		}
	}
}

func (h *wsHandler) Read(c *ws.Conn) {
	defer func() {
		close(c.Done)
		c.Close()
	}()

	for {
		_, received, err := c.ReadMessage()
		if err == nil {
			h.receivedMessages <- string(received)
		} else {
			h.receivedErrors <- err.Error()
		}
	}
}
