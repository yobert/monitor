Minimalist monitoring with PagerDuty.

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

You can also use the very cool free service [https://ntfy.sh/](ntfy) if all you need is push notifications:

    [
        {"url": "https://www.google.com/"},
        {"pagerduty": "081ecc5e6dd6ba0d150fc4bc0e62ec50", "url": "https://www.mysite.com/"},
        {"ntfy: "mypersonalalerts", "url": "https://mypersonalsite.com/"}
    ]

Building and running
--------------------
Requires working [https://golang.org](Go environment).

    go build
    ./monitor

