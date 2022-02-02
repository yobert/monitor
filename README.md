Minimalist monitoring with PagerDuty and ntfy.sh.

Configuration
-------------
Create a services.json. The only required service field is URL:

    [
        {"url": "https://www.google.com/"}
    ]

To connect to pagerduty, create a service and an integration with PagerDuty's events API v2, and paste the API key:

    [
        {"url": "https://www.google.com/"},
        {"pagerduty": "081ecc5e6dd6ba0d150fc4bc0e62ec50", "url": "https://www.mysite.com/"}
    ]

You can also use the very cool free service [ntfy](https://ntfy.sh/) if all you need is push notifications:

    [
        {"url": "https://www.google.com/"},
        {"pagerduty": "081ecc5e6dd6ba0d150fc4bc0e62ec50", "url": "https://www.mysite.com/"},
        {"ntfy: "mypersonalalerts", "url": "https://mypersonalsite.com/"}
    ]

Building and running
--------------------
Requires working [Go environment](https://golang.org).

    go build
    ./monitor

