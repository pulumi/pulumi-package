import * as pkg from "@pulumi/pulumi-package";

const page = new pkg.StaticPage("page", {
    indexContent: "<html><body><p>Hello world!</p></body></html>",
});

export const bucket = page.bucket;
export const url = page.websiteUrl;
