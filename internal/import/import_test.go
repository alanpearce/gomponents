package import_test

import (
	"testing"

	. "alin.ovh/gomponents"
	. "alin.ovh/gomponents/components"
	. "alin.ovh/gomponents/html"
	. "alin.ovh/gomponents/http"
)

func TestImports(t *testing.T) {
	t.Run(
		"this is just a test that does nothing, but I need the dot imports above",
		func(t *testing.T) {
			_ = El("div")
			_ = A()
			_ = HTML5(HTML5Props{})
			_ = Adapt(nil)
		},
	)
}
