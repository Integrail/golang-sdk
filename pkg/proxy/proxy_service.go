package proxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/simple-container-com/go-aws-lambda-sdk/pkg/logger"
	"github.com/simple-container-com/go-aws-lambda-sdk/pkg/service"
	"golang.org/x/sync/errgroup"
)

type StatusResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type Service interface {
	Start(ctx context.Context)
	IsAlive() bool
	Host() *string
}

type checker struct {
	log            logger.Logger
	proxyHost      string
	proxyUsername  string
	proxyPassword  string
	isAlive        *atomic.Bool
	localProxyHost string
	isDebug        bool
	parsedProxyURL *url.URL
}

func NewService(proxyHost string, log logger.Logger, isDebug bool) (Service, error) {
	isAlive := &atomic.Bool{}
	isAlive.Store(true)
	c := &checker{
		log:       log,
		proxyHost: proxyHost,
		isAlive:   isAlive,
		isDebug:   isDebug,
	}
	if parsedURL, err := url.Parse(proxyHost); err != nil {
		return nil, errors.Wrapf(err, "failed to parse proxy host")
	} else if parsedURL.User != nil {
		c.parsedProxyURL = parsedURL
		c.proxyUsername = parsedURL.User.Username()
		c.proxyPassword, _ = parsedURL.User.Password()
	}
	return c, nil
}

func (c *checker) Host() *string {
	if c.IsAlive() && c.localProxyHost != "" {
		return lo.ToPtr(c.localProxyHost)
	} else {
		return nil
	}
}

func (c *checker) Start(ctx context.Context) {
	errG, ctx := errgroup.WithContext(ctx)
	errG.Go(func() error {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if err := c.checkAlive(ctx); err != nil {
					c.log.Errorf(c.log.WithValue(ctx, "error", err.Error()), "integrail proxy error")
					c.isAlive.Store(false)
				} else {
					c.isAlive.Store(true)
				}
			}
		}
	})

	if c.proxyUsername != "" && c.proxyPassword != "" {
		errG.Go(func() error {
			httpProxy := goproxy.NewProxyHttpServer()
			httpProxy.Verbose = c.isDebug
			listener, err := net.Listen("tcp", ":0")
			if err != nil {
				c.log.Errorf(c.log.WithValue(ctx, "error", err.Error()), "failed to get free port")
				return errors.Wrapf(err, "failed to get free port")
			}
			httpProxy.ConnectDial = httpProxy.NewConnectDialToProxyWithHandler(c.proxyHost, func(req *http.Request) {
				authorization := c.proxyAuthorization()
				req.Header.Set("Proxy-Authorization", authorization)
			})
			if c.parsedProxyURL != nil {
				httpProxy.Tr = &http.Transport{Proxy: http.ProxyURL(c.parsedProxyURL)}
			}
			httpProxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				authorization := c.proxyAuthorization()
				req.Header.Set("Proxy-Authorization", authorization)
				return req, nil
			})
			freePort := listener.Addr().(*net.TCPAddr).Port
			_ = listener.Close()
			c.localProxyHost = fmt.Sprintf("localhost:%d", freePort)
			c.log.Infof(ctx, "starting local proxy server at %s", c.localProxyHost)
			if err := http.ListenAndServe(c.localProxyHost, httpProxy); err != nil {
				c.log.Errorf(c.log.WithValue(ctx, "error", err.Error()), "failed to start proxy-proxy-proxy server")
				return errors.Wrapf(err, "failed to start proxy for proxy for proxy")
			}
			return nil
		})
	}
}

func (c *checker) IsAlive() bool {
	return c.isAlive.Load()
}

func (c *checker) checkAlive(ctx context.Context) error {
	client := &http.Client{Timeout: time.Second * 8}

	proxyHost := c.proxyHost
	if !strings.HasPrefix(c.proxyHost, "http") {
		// assuming we're using http proxy
		proxyHost = fmt.Sprintf("http://%s", c.proxyHost)
	}

	statusUrl := fmt.Sprintf("%s/proxy/status", proxyHost)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusUrl, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to init request")
	}
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to check status of proxy")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to check status of agent: status code %d: %s", resp.StatusCode, string(service.ReadBytes(resp.Body)))
	}

	var agentResponse StatusResponse
	respBytes := service.ReadBytes(resp.Body)
	err = json.Unmarshal(respBytes, &agentResponse)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal baas response")
	}

	return nil
}

func (c *checker) proxyAuthorization() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.proxyUsername, c.proxyPassword)))
}
