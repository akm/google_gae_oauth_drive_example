# Google GAE Oauth Drive example

## APIs

Enable these APIｓ at https://console.cloud.google.com/apis/dashboard

- Google Calendar API

## Generate OAuth client

Generate OAuth client at https://console.cloud.google.com/apis/credentials

1. Click `Create credentials`
2. Click 'OAuth client ID'
3. Select `Web application` for `Application type`
4. Enter `google-gae-oauth-drive-example-client1` for `Nmae`

## Deploy App

```
$ export PROJECT=xxxxxx
$ make deploy
```

Check a version deployed.
https://console.cloud.google.com/appengine/versions?serviceId=google-gae-oauth-drive-example&versionssize=50

## Set callback URL to generated OAuth client ID

https://[host name of App]/oauth2callback
