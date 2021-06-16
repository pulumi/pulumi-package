package invoke

import (
	"context"
	"os"
	"os/exec"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

var New = Invoke{
	TypeToken: "pulumi-package:index:New",
	InvokeF: func(args resource.PropertyMap) (map[string]interface{}, error) {
		language := args["language"].StringValue()
		packageName := args["name"].StringValue()
		ctx := context.Background()
		// for now, just call platypack
		// when we do this for real, the templates will be built into the provider binary via go:embed
		cmd := exec.CommandContext(ctx, "platypack", "new", language, packageName)
		cmd.Dir = ""
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return nil, err
	},
}
