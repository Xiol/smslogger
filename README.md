SMS Logger
==========

In Tasker on Android you can configure it to make a HTTP POST request on an incoming SMS message (see screenshots in tasker\_setup for examples of how I did it). This thing accepts those requests and logs the SMS to a database for later retrieval.

The code here is likely terrible. This has been thrown together pretty quickly. The web interface has also been designed so I don't have to write a single line of JS, because why would I do that to myself?

Expects PostgreSQL as a database backend but shouldn't be too hard to make it work with another database.
