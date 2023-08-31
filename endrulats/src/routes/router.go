package routes

import (
	"fmt"
	"net/http"

	"example.com/endrulats/src/handler/get/home"
	"example.com/endrulats/src/handler/get/loginemail"
	"example.com/endrulats/src/handler/post/createuser"
	"example.com/endrulats/src/handler/restful/post"
	"github.com/labstack/echo/v4"
)

func Routes(e *echo.Echo) {

	//get
	e.GET("/home", home.Home)
	e.GET("/loginemail/:email/:sitetoken", loginemail.LoginEmail)
	e.GET("/request", func(c echo.Context) error {
		req := c.Request()
		format := `
			<code>
				Protocol: %s<br>
				Host: %s<br>
				Remote Address: %s<br>
				Method: %s<br>
				Path: %s<br>
			</code>
		`
		return c.HTML(http.StatusOK, fmt.Sprintf(format, req.Proto, req.Host, req.RemoteAddr, req.Method, req.URL.Path))
	})

	//post
	e.POST("/usercreate", createuser.Createuser)
	e.POST("/p", post.Posts)

}
