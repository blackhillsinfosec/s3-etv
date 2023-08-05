package main

import (
    "github.com/spf13/cobra"
)

var (
    rootCmd = &cobra.Command{
        Use:   "--s3-object-url HTTPS_URL --input-file FILE_PATH",
        Short: "Validate integrity of a downloaded S3 object file.",
        Long: "Verify integrity of a downloaded S3 object by generating an ETag and comparing\n" +
            "it to a value retrieved from the S3 API.\n\n" +
            "- To supply credentials, see \"Specifying Credentials\" at:\n" +
            "    - https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#Spedifying-Credentials\n" +
            "- For further reading on ETags, see \"Using part-level checksums for multipart uploads\":\n" +
            "    - https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html",
        Example: "--s3-object-url https://my-bucket.s3.amazonaws.com/a-big-file.iso --input-file a-big-file.iso",
        Run:     func(_ *cobra.Command, _ []string) { validate() },
    }
)

func init() {
    rootCmd.PersistentFlags().StringVarP(&inputFile, "input-file", "i", "", "Downloaded S3 object file.")
    rootCmd.PersistentFlags().StringVarP(&s3ObjectUrl, "s3-object-url", "u", "", "HTTP URL to the object.")
    rootCmd.PersistentFlags().StringVarP(&s3Region, "s3-region", "r", "us-east-1", "Region of the S3 bucket.")
    rootCmd.MarkPersistentFlagRequired("input-file")
    rootCmd.MarkPersistentFlagRequired("s3-object-url")
}
