FuellyView
==========

This project is a Google App Engine app designed to scrape <a href="http://www.fuelly.com">Fuelly</a> and redisplay the data.  I wanted a way to view the data on Fuelly sorted by fuel effeciency and since they have decided against that (http://www.fuelly.com/faq/13/order-of-best-fuel-economy) alternative measures had to be taken.

Go is the backend language used with Angularjs being used on the frontend.

Here are the main two urls hosted:
* / or /index.html - This is the main view
* /refresh - This initiates the scraping of www.fuelly.com

### Note:
* At this time the scraping is not pulling all of the cars.  Need to figure out why.
* If you are using a free GAE account the /refresh WILL use up all of your datastore writes for the day.