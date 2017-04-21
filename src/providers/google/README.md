# google

# Create

There are a few steps to take before hosting your static website with google:

1. Install [gcloud](https://cloud.google.com/sdk/downloads)

* Enable API access
  * For the [google cloud storage api](https://console.cloud.google.com/apis/api/storage-component.googleapis.com/)
  * And the [google cloud storage json api]( https://console.developers.google.com/apis/api/storage_api/overview)

* Log in using `gcloud auth application-default login` on the command line

* [Verify your domain](https://www.google.com/webmasters/verification/home)
  * If you'd like, you can use the `--domain-validation-help` flag with `stout create`, and have stout prompt you to type in a record type and value.
    * Click on the alternate domain methods tab, select other, and use DNS validation (either TXT or CNAME records)
    * In stout, type in the record type (`CNAME` or `TXT`)
    * Then, type in the record value that google asks you to use

## Options

The `google-project-id` string flag holds the project ID, which you can find [here](https://console.cloud.google.com/iam-admin/settings/project).

## Config

The providers section of an example config file using google could look like the following:

```yaml
[...]
        providers:
                google:
                        project-id: test-example-1239234
```
