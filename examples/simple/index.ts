import * as pkg from "@pulumi/pulumipackage";
import * as aws from "@pulumi/aws";

const bucket = new aws.s3.Bucket("serverBucket", {
    website: {
        indexDocument: "index.html",
    },
    forceDestroy: true,
});

new aws.s3.BucketPolicy("bucketPolicy", {
    bucket: bucket.bucket,
    policy: bucket.bucket.apply(name => JSON.stringify({
        Version: "2012-10-17",
        Statement: [
            {
                Effect: "Allow",
                Principal: "*",
                Action: ["s3:GetObject"],
                Resource: [
                    `arn:aws:s3:::${name}/*`, // policy refers to bucket name explicitly
                ],
            },
        ],
    })),
}, {
    parent: bucket,
});

export const pulumiPackage = new pkg.Package("pkg", {
    language: "go",
    name: "pulumiception",
    serverBucketName: bucket.bucket,
    serverBucketWebsiteEndpoint: bucket.websiteEndpoint
});
