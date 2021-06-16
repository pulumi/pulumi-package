package invoke

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

var Publish = Invoke{
	TypeToken: "pulumi-package:index:Publish",
	InvokeF: func(args resource.PropertyMap) (map[string]interface{}, error) {
		bucketName := args["serverBucketName"].StringValue()
		webEndpoint := args["serverBucketWebsiteEndpoint"].StringValue()
		version := args["version"].StringValue()
		name := args["name"].StringValue()
		pluginBinary := fmt.Sprintf("pulumi-resource-%s", name)
		pluginArchive := fmt.Sprintf("pulumi-resource-%s-%s-darwin-amd64.tar.gz", name, version)

		ctx := context.Background()
		cmd := exec.CommandContext(ctx, "tar", "-zcvf", pluginArchive, "-C", "bin", pluginBinary)
		cmd.Dir = ""
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return nil, err
		}

		cmd = exec.CommandContext(ctx, "aws", "s3", "cp", pluginArchive, fmt.Sprintf("s3://%s", bucketName))
		cmd.Dir = ""
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"url": fmt.Sprintf("%v/%v", webEndpoint, pluginArchive),
		}, nil
	},
}
