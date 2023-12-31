package server

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"example.com/endrulats/assets"
	"example.com/endrulats/src/funcmaps"
	"example.com/endrulats/src/routes"

	"github.com/Masterminds/sprig/v3"

	"example.com/endrulats/internal/dbsql/user"
	"example.com/endrulats/src/handler/get/welcome"

	"example.com/endrulats/internal/security/tokens"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func Server() {

	e := echo.New()
	files, err := getAllFilenames(&assets.Assets)
	if err != nil {
		fmt.Print(err)
	}

	//if you are planning on using binary assets then use this but if you turn it on then
	//you have to rebuild every time you rerun it.
	// filesoptimized, err := getAllFilenames(&assets.AssetsOptimized)
	// if err != nil {
	// 	fmt.Print(err)
	// }

	//for CSP policy to ensure that the assets are always available and secure
	Nonce := fmt.Sprintf("nonce-%d", 1)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("n", Nonce)
			return next(c)
		}
	})

	renderer := &TemplateRenderer{
		templates: template.Must(template.New("t").Funcs(template.FuncMap{
			"IndexCount":     funcmaps.IndexCount,
			"RemoveBrackets": funcmaps.RemoveBrackets,
		}).Funcs(sprig.FuncMap()).ParseFS(assets.Assets, files...)),
	}

	e.Renderer = renderer

	queryAuthConfig := middleware.KeyAuthConfig{
		KeyLookup: "query:sitetoken,header:headkey,cookie:goservershell",
		Validator: func(key string, c echo.Context) (bool, error) {
			user := new(user.Users)
			email := c.Param("email")
			idkey := c.Param("sitetoken")

			err, exists := user.CheckUser(c, email, idkey)
			if err != nil {
				fmt.Println("middleware", exists)
				return false, err
			}

			fmt.Println(key, " keylookup")
			b := tokens.Checktokencontext(key)
			return b, nil
		},

		ErrorHandler: func(error, echo.Context) error {
			var err error

			return err
		},
	}
	r := e.Group("/restricted")
	r.Use(middleware.KeyAuthWithConfig(queryAuthConfig))
	r.GET("/welcome/:email/:sitetoken", welcome.Welcome)

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: getFileSystem(assets.Assets),
		HTML5:      true,
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	routes.Routes(e)

	// Route
	e.Logger.SetLevel(log.ERROR)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// Generate a nonce

	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'; style-src 'self' 'nonce-" + Nonce + "'",
	}))

	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "header:headkey",
		CookiePath:     "/",
		CookieDomain:   "localhost",
		CookieSecure:   true,
		CookieHTTPOnly: true,
	}))
	e.Use(middleware.BodyLimit("3M"))
	e.IPExtractor = echo.ExtractIPDirect()
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(30)))
	e.Static("/", "assets/optimized")

	e.Logger.Fatal(e.StartTLS(":5002", "cert.pem", "key.pem"))

}

func GetAllFilePathsInDirectory(dirpath string) ([]string, error) {
	var paths []string
	err := filepath.Walk(dirpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func ParseDirectory(dirpath string) (*template.Template, error) {
	paths, err := GetAllFilePathsInDirectory(dirpath)
	if err != nil {
		return nil, err
	}
	return template.ParseFiles(paths...)
}

type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {

	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

var err error

func getFileSystem(TmplMainGo embed.FS) http.FileSystem {

	log.Print("using embed mode")
	fsys, err := fs.Sub(TmplMainGo, "assets/templates")
	if err != nil {
		log.Print(err)
	}

	return http.FS(fsys)
}

// https://gist.github.com/clarkmcc/1fdab4472283bb68464d066d6b4169bc
func getAllFilenames(efs *embed.FS) (files []string, err error) {
	if err := fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}
