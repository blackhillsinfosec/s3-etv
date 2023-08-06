# S3 ETag Validator

During upload, Amazon S3 generates and assigns server-side ETag values that can be used
to validate the integrity of downloaded S3 objects. `s3-etv` performs validation by
retrieving various object attributes from the S3 API and implementing the same algorithm
locally to produce an ETag value that is then compared to the server-side value.

Download binaries from the [releases page](https://github.com/blackhillsinfosec/s3-etv/releases/).

# Usage

0. Download `s3-etv` from the [releases page](https://github.com/blackhillsinfosec/s3-etv/releases/).
1. Download the target file from AWS S3.
   - _Tip_: Retain the URL as it will be needed later.
2. Run `s3-etv` to validate integrity of the downloaded file:

```
./s3-etv -u https://some-bucket.s3.amazonaws.com/some.vhd  -i ~/Downloads/some.vhd --thread-count 15
[INF] 2023/08/06 06:24:02 Retrieved ETag information via S3 API
[INF] 2023/08/06 06:24:02 ETag Value: fcd95600383564e0916872202cda9c3d-2185
[INF] 2023/08/06 06:24:02 Input File: /home/user/Downloads/some.vhd
[WRN] 2023/08/06 06:24:02 Verifying file integrity. This will take some time.
[WRN] 2023/08/06 06:24:02 Using process count: 15
[INF] 2023/08/06 06:24:04 Integrity check finished
[INF] 2023/08/06 06:24:04 AWS ETag        > fcd95600383564e0916872202cda9c3d-2185
[INF] 2023/08/06 06:24:04 Calculated ETag > fcd95600383564e0916872202cda9c3d-2185
[INF] 2023/08/06 06:24:04 ETag values matched
[INF] 2023/08/06 06:24:04 File integrity status: confirmed
[INF] 2023/08/06 06:24:04 Verification Duration: 0:0:1.1406

File integrity: verified
```

When the integrity check fails, `File integrity: compromised` is returned.

# FAQ

## Why is this a thing?

We get it, it isn't a very hackery tool. But we often share large files via S3 to customers
to facilitate remote access during engagements and this approach provides multi-OS support with
multiprocessing capabilities while avoiding garbage like PowerShell.

## How does AWS S3 Generate ETags?

[AWS says...](https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html?icmpid=docs_amazons3_console#checking-object-integrity-etag-and-md5)