SMS Logger
==========

In Tasker on Android you can configure it to make a HTTP POST request on an incoming SMS message. This thing accepts those requests and logs the SMS to a database for later retrieval.

The code here is likely terrible. This has been thrown together pretty quickly. The web interface has also been designed so I don't have to write a single line of JS, because why would I do that to myself?

Expects PostgreSQL as a database backend but shouldn't be too hard to make it work with another database.

Tasker Setup
------------

Create a task when an SMS message comes in to do the following steps:

> A1: Variable Set [ Name:%smsbody To:%SMSRB Do Maths:Off Append:Off ] 
> A2: Variable Convert [ Name:%smsbody Function:Base64 Encode Store Result In:%smsbody ] 
> A3: Variable Search Replace [ Variable:%smsbody Search:\n Ignore Case:On Multi-Line:Off One Match Only:Off Store Matches In: Replace Matches:On Replace With: ] 
> A4: HTTP Post [ Server:Port:https://user:password@logging.host.com Path:/sms Data / File:message=%smsbody
> date=%SMSRD
> time=%SMSRT
> from=%SMSRN Cookies: User Agent: Timeout:30 Content Type:application/x-www-form-urlencoded Output File: Trust Any Certificate:Off Continue Task After Error:On ] 
> A5: Wait [ MS:0 Seconds:10 Minutes:0 Hours:0 Days:0 ] If [ %HTTPR neq 201 ]
> A6: Variable Add [ Name:%retries Value:1 Wrap Around:0 ] If [ %HTTPR neq 201 ]
> A7: Goto [ Type:Action Label Number:1 Label:Start ] If [ %HTTPR neq 201 & %retries < 6 ]
> A8: Notify [ Title:SMS Failed To Post Text:Last error: %HTTPR %HTTPD Icon:cust\_warning Number:0 Permanent:Off Priority:3 ] If [ %HTTPR neq 201 ]

