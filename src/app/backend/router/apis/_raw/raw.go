package _raw

/**
  참조
    https://kubernetes.io/docs/reference/using-api/api-concepts/
    https://github.com/gin-gonic/gin
*/

import (
	"net/http"
	"strings"

	"github.com/acornsoftlab/dashboard/pkg/app"
	"github.com/acornsoftlab/dashboard/pkg/client"
	"github.com/acornsoftlab/dashboard/pkg/config"
	"github.com/acornsoftlab/dashboard/pkg/lang"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
)

// Get api group list
func GetAPIGroupList(c *gin.Context) {
	g := app.Gin{C: c}

	// instancing dynamic client
	context := lang.NVL(g.C.Param("CLUSTER"), config.Value.CurrentContext)
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config.Value.KubeConfigs[context])
	if err != nil {
		g.SendMessage(http.StatusInternalServerError, err.Error())
		return
	}

	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		g.SendMessage(http.StatusInternalServerError, err.Error())
		return
	}

	g.Send(http.StatusOK, groups)

}

// Create or Update
func ApplyRaw(c *gin.Context) {
	g := app.Gin{C: c}

	// api clinet
	context := lang.NVL(g.C.Param("CLUSTER"), config.Value.CurrentContext)
	api := client.NewDynamicClient(config.Value.KubeConfigs[context])

	// invoke POST
	r, err := api.POST(g.C.Request.Body, g.C.Request.Method == "PUT")
	if err != nil {
		g.SendMessage(http.StatusBadRequest, err.Error())
		return
	}

	g.Send(http.StatusCreated, r)
}

// Delete
func DeleteRaw(c *gin.Context) {
	g := app.Gin{C: c}

	// url parameter validation
	v := []string{"VERSION", "RESOURCE", "NAME"}
	if err := g.ValidateUrl(v); err != nil {
		g.SendMessage(http.StatusBadRequest, err.Error())
		return
	}

	// instancing dynamic client
	context := lang.NVL(g.C.Param("CLUSTER"), config.Value.CurrentContext)
	api := client.NewDynamicClientSchema(config.Value.KubeConfigs[context], c.Param("GROUP"), c.Param("VERSION"), c.Param("RESOURCE"))
	api.SetNamespace(c.Param("NAMESPACE"))

	// invoke delete
	if err := api.DELETE(c.Param("NAME"), v1.DeleteOptions{}); err != nil {
		g.SendMessage(http.StatusInternalServerError, err.Error())
		return
	}

}

// Get or List
func GetRaw(c *gin.Context) {
	g := app.Gin{C: c}

	// instancing dynamic client
	context := lang.NVL(g.C.Param("CLUSTER"), config.Value.CurrentContext)
	api := client.NewDynamicClientSchema(config.Value.KubeConfigs[context], c.Param("GROUP"), c.Param("VERSION"), c.Param("RESOURCE"))
	api.SetNamespace(c.Param("NAMESPACE"))

	var r interface{}
	var err error

	if c.Param("NAME") == "" {
		r, err = api.List(v1.ListOptions{})
		if err != nil {
			g.SendMessage(http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		r, err = api.GET(c.Param("NAME"), v1.GetOptions{})
		if err != nil {
			if strings.HasSuffix(err.Error(), "not found") {
				g.SendMessage(http.StatusNotFound, err.Error())
			} else {
				g.SendMessage(http.StatusInternalServerError, err.Error())
			}
			return
		}
	}

	g.Send(http.StatusOK, r)

}

// Patch
func PatchRaw(c *gin.Context) {
	g := app.Gin{C: c}

	// url parameter validation
	v := []string{"VERSION", "RESOURCE", "NAME"}
	if err := g.ValidateUrl(v); err != nil {
		g.SendMessage(http.StatusBadRequest, err.Error())
		return
	}

	// instancing dynamic client
	context := lang.NVL(g.C.Param("CLUSTER"), config.Value.CurrentContext)
	api := client.NewDynamicClientSchema(config.Value.KubeConfigs[context], c.Param("GROUP"), c.Param("VERSION"), c.Param("RESOURCE"))
	api.SetNamespace(c.Param("NAMESPACE"))

	var r interface{}
	var err error

	r, err = api.PATCH(c.Param("NAME"), types.PatchType(c.ContentType()), c.Request.Body, v1.PatchOptions{})
	if err != nil {
		if strings.HasSuffix(err.Error(), "not found") {
			g.SendMessage(http.StatusNotFound, err.Error())
		} else {
			g.SendMessage(http.StatusInternalServerError, err.Error())
		}
		return
	}

	g.Send(http.StatusOK, r)

}
