# google

## Getting Started 

There are a few steps to take before hosting your static website with Google. Some of these steps can be skipped if you're only using Google for DNS or CDN.

1. Create a service account:
    * Navigate to the [iam & admin -> service accounts](https://console.cloud.google.com/iam-admin/serviceaccounts) section.
        * If navigating through this link, also select your project.
    * Create a service account with role set to *Storage Admin*.
    * Save the json auth file.

1. [Verify your domain](https://www.google.com/webmasters/verification/home) ([official instructions here](https://support.google.com/a/answer/183895?hl=en))
    * Click on the alternate domain methods tab, select other, and use DNS validation (either `TXT` or `CNAME` records). This option may also be called "Domain name provider: sign in to your domain name provider".
    * If you'd like, you can use the `--domain-validation-help` flag with `stout create`, and have stout prompt you to type in a record type and value.
        * In stout, type in the record type (`CNAME` or `TXT`).
        * Then, type in the record value that google asks you to use.

1. You're ready to go!

## Options

The `google-project-id` string flag holds the project ID, which you can find [here](https://console.cloud.google.com/iam-admin/settings/project).

## Config

The providers section of an example config file using google could look like the following:

```yaml
[...]
    providers:
        google:
            keyfile: './google-auth.json'
            project-id: test-example-1239234
```
