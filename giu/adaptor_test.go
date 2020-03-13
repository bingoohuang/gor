// nolint gomnd
package giu_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bingoohuang/gor"

	"github.com/bingoohuang/gor/giu"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type Rsp struct {
	State int
	Data  interface{}
}

type SetAgeReq struct {
	Name string
	Age  int
}

type SetAgeRsp struct {
	Name string
}

type AuthUser struct {
	Name string
}

func TestUMP(t *testing.T) {
	ga := giu.NewAdaptor()

	// 注册如何处理成功返回一个值
	ga.RegisterSuccProcessor(func(c *gin.Context, vs ...interface{}) {
		if len(vs) == 0 {
			c.JSON(http.StatusOK, Rsp{State: http.StatusOK, Data: "ok"}) // 如何处理无返回(单独error返回除外)
		} else if rsp, ok := vs[0].(Rsp); ok { // 返回已经是Rsp类型，不再包装
			c.JSON(http.StatusOK, rsp)
		} else {
			c.JSON(http.StatusOK, Rsp{State: http.StatusOK, Data: vs[0]}) // 选取第一个返回参数，JSON返回
		}
	})

	// 注册如何处理错误
	ga.RegisterErrProcessor(func(c *gin.Context, vs ...interface{}) {
		c.JSON(http.StatusOK, Rsp{State: http.StatusInternalServerError, Data: vs[0].(error).Error()})
	})

	// 注册如何处理AuthUser类型的输入参数
	ga.RegisterTypeProcessor(AuthUser{}, func(c *gin.Context, vs ...interface{}) (interface{}, error) {
		return gor.V0(c.Get("AuthUser")), nil
	})

	resp := httptest.NewRecorder()
	c, r := gin.CreateTestContext(resp)

	ptrAuthUser := true
	r.Use(func(c *gin.Context) {
		if ptrAuthUser {
			c.Set("AuthUser", &AuthUser{Name: "TestAuthUser"})
		} else {
			c.Set("AuthUser", AuthUser{Name: "TestAuthUser"})
		}

		ptrAuthUser = !ptrAuthUser
	})

	type MyObject struct {
		Name string
	}

	gr := ga.Route(r)
	gr.Use(func() *MyObject { return &MyObject{Name: "Test"} })

	gr.GET("/MyObject1", func(m MyObject) string { return m.Name })
	gr.GET("/MyObject2", func(m *MyObject) string { return m.Name })

	gr.GET("/GetAge1/:name", func(user AuthUser, name string) string {
		return user.Name + "/" + name
	}, giu.Params(giu.URLParam("name")))
	gr.GET("/GetAge2/:name", func(name string, user AuthUser) string {
		return user.Name + "/" + name
	}, giu.Params(giu.URLParam("name")))
	gr.GET("/GetAge3/:name", func(user *AuthUser, name string) string {
		return user.Name + "/" + name
	}, giu.Params(giu.URLParam("name")))
	gr.GET("/GetAge4/:name", func(name string, user *AuthUser) string {
		return user.Name + "/" + name
	}, giu.Params(giu.URLParam("name")))
	gr.POST("/SetAge", func(req SetAgeReq) SetAgeRsp {
		return SetAgeRsp{Name: fmt.Sprintf("%s:%d", req.Name, req.Age)}
	})
	gr.GET("/Get/:name/:age", func(name string, age int) (Rsp, error) {
		return Rsp{State: 200, Data: fmt.Sprintf("%s:%d", name, age)}, nil
	}, giu.Params(giu.URLParam("name"), giu.URLParam("age")))
	gr.Any("/error", func() error { return errors.New("error occurred") })
	gr.GET("/ok", func() error { return nil })
	gr.GET("/url", func(c *gin.Context) string { return c.Request.URL.String() })

	checkStatusOK(t, resp, c, r, "/GetAge1/bingoo", "TestAuthUser/bingoo")
	checkStatusOK(t, resp, c, r, "/GetAge2/bingoo", "TestAuthUser/bingoo")
	checkStatusOK(t, resp, c, r, "/GetAge3/bingoo", "TestAuthUser/bingoo")
	checkStatusOK(t, resp, c, r, "/GetAge4/bingoo", "TestAuthUser/bingoo")
	checkBody(t, resp, c, r, http.MethodPost, "/SetAge", 200,
		SetAgeReq{Name: "bingoo", Age: 100}, SetAgeRsp{Name: "bingoo:100"})
	check(t, resp, c, r, "/error", 500, "error occurred")
	checkStatusOK(t, resp, c, r, "/ok", "ok")
	checkStatusOK(t, resp, c, r, "/Get/bingoo/100", "bingoo:100")
	checkStatusOK(t, resp, c, r, "/url", "/url")
	checkStatusOK(t, resp, c, r, "/MyObject1", "Test")
	checkStatusOK(t, resp, c, r, "/MyObject2", "Test")
}

func checkStatusOK(t *testing.T, rr *httptest.ResponseRecorder, c *gin.Context, r *gin.Engine, url string, d interface{}) {
	check(t, rr, c, r, url, http.StatusOK, d)
}

func check(t *testing.T, rr *httptest.ResponseRecorder, c *gin.Context, r *gin.Engine, url string, state int, d interface{}) {
	checkBody(t, rr, c, r, http.MethodGet, url, state, nil, d)
}
func checkBody(t *testing.T, rr *httptest.ResponseRecorder, c *gin.Context, r *gin.Engine,
	method, url string, state int, b interface{}, d interface{}) {
	if b != nil {
		bb, _ := json.Marshal(b)
		c.Request, _ = http.NewRequest(method, url, bytes.NewReader(bb))
		c.Request.Header.Set("Content-Type", "application/json; charset=utf-8")
	} else {
		c.Request, _ = http.NewRequest(method, url, nil)
	}

	r.ServeHTTP(rr, c.Request)
	assert.Equal(t, http.StatusOK, rr.Code)
	rsp, _ := json.Marshal(Rsp{State: state, Data: d})
	body, _ := ioutil.ReadAll(rr.Body)
	assert.Equal(t, string(rsp), strings.TrimSpace(string(body)))
}

func TestHello(t *testing.T) {
	router := gin.New()

	hello := ""
	world := ""

	ga := giu.NewAdaptor()
	gr := ga.Route(router)

	gr.GET("/hello/:arg", func(v string) { hello = v }, giu.Params(giu.URLParam("arg")))
	gr.GET("/world", func(v string) string {
		world = v
		return "hello " + v
	}, giu.Params(giu.QueryParam("arg")))

	gr.GET("/error", func() error {
		return errors.New("xxx")
	})

	rr := performRequest("GET", "/hello/bingoo", router)

	assert.Equal(t, "bingoo", hello)
	assert.Equal(t, gor.V([]byte{}, nil), gor.V(ioutil.ReadAll(rr.Body)))

	rr = performRequest("GET", "/world?arg=huang", router)

	assert.Equal(t, "huang", world)
	bytes, _ := ioutil.ReadAll(rr.Body)
	assert.Equal(t, `hello huang`, strings.TrimSpace(string(bytes)))

	rr = performRequest("GET", "/error", router)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// from https://github.com/gin-gonic/gin/issues/1120
func performRequest(method, target string, router *gin.Engine) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}