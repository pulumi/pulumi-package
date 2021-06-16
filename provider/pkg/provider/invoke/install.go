package invoke

import (
	"context"
	"os"
	"os/exec"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

var Install = Invoke{
	TypeToken: "pulumi-package:index:Install",
	InvokeF: func(args resource.PropertyMap) (map[string]interface{}, error) {
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, "make", "install_provider", "install_nodejs_sdk")
		cmd.Dir = ""
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return nil, err
	},
}
