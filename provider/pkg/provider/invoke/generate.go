package invoke

import (
	"context"
	"os"
	"os/exec"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

var Generate = Invoke{
	TypeToken: "pulumi-package:index:Generate",
	InvokeF: func(args resource.PropertyMap) (map[string]interface{}, error) {
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, "make", "generate")
		cmd.Dir = ""
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return nil, err
	},
}
