---
title: proto/txnotify.proto version not set
language_tabs:
  - python: Python
language_clients:
  - python: ""
toc_footers: []
includes: []
search: false
highlight_theme: darkula
headingLevel: 2

---

<!-- Generator: Widdershins v4.0.1 -->

<h1 id="proto-txnotify-proto">proto/txnotify.proto version not set</h1>

> Scroll down for code samples, example requests and responses. Select a language for code samples from the tabs above or the mobile navigation menu.

<h1 id="proto-txnotify-proto-notify">Notify</h1>

## ListNotifications can be used to list all your current active notifications

<a id="opIdListNotifications"></a>

> Code samples

```python
import requests
headers = {
  'Accept': 'application/json'
}

r = requests.get('/notifications', headers = headers)

print(r.json())

```

`GET /notifications`

<h3 id="listnotifications-can-be-used-to-list-all-your-current-active-notifications-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|user_id|query|string|false|none|

> Example responses

> 200 Response

```json
{
  "notifications": [
    {
      "user_id": "string",
      "identifier": "string",
      "confirmations": 0,
      "email": "string",
      "description": "string",
      "slack_webhook_url": "string",
      "callback_url": "string"
    }
  ]
}
```

<h3 id="listnotifications-can-be-used-to-list-all-your-current-active-notifications-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A successful response.|[ListNotificationsResponse](#schemalistnotificationsresponse)|

<aside class="success">
This operation does not require authentication
</aside>

## CreateNotification

<a id="opIdCreateNotification"></a>

> Code samples

```python
import requests
headers = {
  'Content-Type': 'application/json',
  'Accept': 'application/json'
}

r = requests.post('/notifications', headers = headers)

print(r.json())

```

`POST /notifications`

> Body parameter

```json
{
  "user_id": "string",
  "identifier": "string",
  "confirmations": 0,
  "email": "string",
  "description": "string",
  "slack_webhook_url": "string",
  "callback_url": "string"
}
```

<h3 id="createnotification-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[Notification](#schemanotification)|true|none|

> Example responses

> 200 Response

```json
{
  "id": "string"
}
```

<h3 id="createnotification-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A successful response.|[CreateNotificationResponse](#schemacreatenotificationresponse)|

<aside class="success">
This operation does not require authentication
</aside>

<h1 id="proto-txnotify-proto-user">User</h1>

## CreateUser

<a id="opIdCreateUser"></a>

> Code samples

```python
import requests
headers = {
  'Accept': 'application/json'
}

r = requests.post('/users', headers = headers)

print(r.json())

```

`POST /users`

> Example responses

> 200 Response

```json
{
  "id": "string"
}
```

<h3 id="createuser-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A successful response.|[CreateUserResponse](#schemacreateuserresponse)|

<aside class="success">
This operation does not require authentication
</aside>

# Schemas

<h2 id="tocS_CreateNotificationResponse">CreateNotificationResponse</h2>
<!-- backwards compatibility -->
<a id="schemacreatenotificationresponse"></a>
<a id="schema_CreateNotificationResponse"></a>
<a id="tocScreatenotificationresponse"></a>
<a id="tocscreatenotificationresponse"></a>

```json
{
  "id": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|false|none|the id of your notification. Can be used to get more specific information about your subscription,<br>or to delete it.|

<h2 id="tocS_CreateUserResponse">CreateUserResponse</h2>
<!-- backwards compatibility -->
<a id="schemacreateuserresponse"></a>
<a id="schema_CreateUserResponse"></a>
<a id="tocScreateuserresponse"></a>
<a id="tocscreateuserresponse"></a>

```json
{
  "id": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|false|none|none|

<h2 id="tocS_ListNotificationsResponse">ListNotificationsResponse</h2>
<!-- backwards compatibility -->
<a id="schemalistnotificationsresponse"></a>
<a id="schema_ListNotificationsResponse"></a>
<a id="tocSlistnotificationsresponse"></a>
<a id="tocslistnotificationsresponse"></a>

```json
{
  "notifications": [
    {
      "user_id": "string",
      "identifier": "string",
      "confirmations": 0,
      "email": "string",
      "description": "string",
      "slack_webhook_url": "string",
      "callback_url": "string"
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|notifications|[[Notification](#schemanotification)]|false|none|none|

<h2 id="tocS_Notification">Notification</h2>
<!-- backwards compatibility -->
<a id="schemanotification"></a>
<a id="schema_Notification"></a>
<a id="tocSnotification"></a>
<a id="tocsnotification"></a>

```json
{
  "user_id": "string",
  "identifier": "string",
  "confirmations": 0,
  "email": "string",
  "description": "string",
  "slack_webhook_url": "string",
  "callback_url": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|user_id|string|false|none|none|
|identifier|string|false|none|The bitcoin blockchain id of the transaction you want to monitor or<br>the bitcoin blockchain address you want to monitor. You will be notified about all<br>new transactions sent to and from this address.|
|confirmations|integer(int64)|false|none|how many confirmations the transaction should have when you want to be notified. Can not be<br>higher than 6. If omitted, you will get a notification at 0 confirmations.|
|email|string|false|none|none|
|description|string|false|none|none|
|slack_webhook_url|string|false|none|none|
|callback_url|string|false|none|none|

