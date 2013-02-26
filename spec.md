Overview
========

Webca is a web application to help you manage and keep track of all those Server Certificates that expire and need to be renewed whenever is most inconvinient for you... or most probably, that you realize they expired only when the browser refuses to load some of your secure pages.

This app let's you create and modify your own CA and server certificates, but the cool thing is that it will also send you an email when the certifcates are about to expire or even renew them for you automatically.

It also lets you import your third party signed certificates to be able to get notifications about them as well.

And all so easy that even a pointy-haired manager could do it.


Scenarios
=========

Joe is a developer and a part-time sysadmin. Most of the time he codes, but sometimes he needs to set up some servers, their backups, updates... and their certificates. Certificates are a pain, for security reasons, they don't last forever and Joe keeps on forgetting when are those certificates gonna expire, he is just too busy coding, most of the time. And certs tend to expire in the worst most busy day.

Marissa is a manager. She used to do some coding, but the truth is that she never liked it too much. But being a manager, dealing with people, giving orders and keep everything under control. She usually lets all technical details to developers, but security is all too important, so she prefers to keep control of certificates herself.
But, hey, anyway she has lots of other things to do apart from keeping track of those certificates, so a little automatic reminder would be helpful, thank you.


Non Goals
=========

Webca es a small one-user app. It will not support multiple user logins. There will be one user in charge for the certificates.

Webca will have few features. Its goal is to provide simple fast & pain-free certificate management.


Complete Usage Scenario
=======================

Joe or Marissa will download the application for their laptop or server. It is advisable that they have secured that machine, specially if the certificates they manage do protect some sensitive data. In any case webca will not provide additional security measures, any user allowed to read the directory in which webca is configured will be able to access the private keys without accessing the webca itself.

The application distribution will include an installation script for all supported OS (Linux, Mac and Windows) and an binary called "webca.binpkg" that cannot be executed unless previously installed by the script. Only the binary renamed different will allow itself to be run.

webca.binpkg, when executed by the script on the command line, will ask for the data directory (the installation directory is the current working directory) and the port. Both values will have defaults:

- The directory will be $HOME/.webca/config.json, or for the root user, the default directory will be /etc/webca, while in Windows it will be %SYSTEMDRIVE%/webca for the Administrator.

- The port will be 8443 by default.

After that the webca.binpkg will generate an setup password to be used afterwards to ensure no one else is setting up the installation, as it can be done remotelly. After that all initial config data will be saved the choosen location to be use by the webapp.

Once the service is setup for the target MAC/Linux or Windows machine, the user will be informed of the URL it is listening on and advised to get a browser pointing there. Joe will go from the chilly server room to his confortable workstation to finish the setup... Marissa, with her laptop, will try to click on the URL to get the browser there and may get piss-off if that doesn't work (not our problem, mind you!), cause she has already spent too much time on the bloody black command line today...

When the user access the correct app URL, it is presented with a small form asking for the setup key. Only the user that knows (or guesses) that key is allowed in. There are just 7 attempts allowed to enter the key correctly, otherwise the system gets blocked and the webapp setup will have to be regenerated locally on the server / target machine (or maybe through ssh in Joe's case). The number of attempts left are advertised after the first failure.

Once the setup password is provided correctly the user is presented with the WebCA Setup Form. This form will ask for the following information:

1. User details
   - __User name__: admin  [type: name; no spaces, a-z,'_' and numbers]
   - User display name:  [type: text]
   - __Password__:       [type: text]
   - __Password confirm__:
   - Email:              [type: email]

2. Certificate Authority
   - __Name__:          [type: name]
   - Address: (Street, Postal Code, Locality, Province, Country)  [type: texts]
   - Org. Unit:         [type: text]
   - __Organization__:	 [type: text]
   - __Duration in Days__: (1825)      [type: DropDown/HTML Select integer representing days]

3. Server Certificate
   - __Server Name__: ("same as the URL, if it's NOT an IP address")   [type: name]
   - Option to change the Address, Org. Unit and Org. to a different value from the CA.
   - __Server Cert duration in Days__: (365)   [type: DropDown/HTML Select integer in days]

Fields with a value mean the user can accept that default value or change it. And bold fields are mandatory. Fields must be correct for their type.

Once the from is correctly filled questions are answered, the user will be able to review all of them and choose to re-edit or confirm and end the setup. Only the first user can complete the setup, the other will get a warning and the transition page...

The transition page comes just after the setup is completed. It advertises the app is ready to be used, and contains a link to download the CA certificate and another to enter the application. The webapp is reset in the background and it restarts with SSL encryption on the server certificate signed by the CA cert.

When entering the configured application the user gets classic login page (login/password+submit). The setup key is long forgotten, so the user needs to use the password chosen in the setup form.

The configured app main page is a list/tree alphabetically ordered CAs with their signed certs, also in alphabetical order. A main page layout example is:

																(Settings)
	(+CA)
			- (CA1) Expiration date (+)
  				- (server0.example.com) Exp. date
  				- (server1.example.com) Exp. date
			- (CA2) (Expiration date) (+)
				- (server2.example.com) Exp. date
				- (server4.example.com) Exp. date
				- (server5.example.com) Exp. date
			- (CA3) (Expiration date) (+)
				- (someotherserver.someotherdomain.com) Exp. date

* Clicking on the listed items you can edit or delete certificates. 
* Links represented here by (+CA) & (+) allow to create more CAs or certs into a CA. 
* Email notifications and user Account details can also be reconfigured using the links in the upper right area of the page.

The use cases, that will be explained in detail later, are:

* Create a new CA
* Create a new Cert within (signed by) an existing CA
* Edit (Download+Renew+Delete+Clone) a CA or Certificate
* Settings

Nevertheless the philosophy of this webapp is that you expend a few minutes installing and configuring it the first time, and later you just forget it is there till there is:
1. The need to create or change certificate.
2. A notification email gets to you cause a Certificate or CA is about to expire.

Fire and forget!... I mean, configure and forget!


Creating a new CA
=================

When clicking on the main page link to create a new CA, (+CA), the CA creation form appears. Its layout is something like this:

	New Certificate Authority...

					[Possible notice]

			Name:
			Address: (Street, Postal Code, Locality, Province, Country)
			Org. Unit:
            Organization:
   			Duration in Days: (1825)

   			(SUBMIT BUTTON)


1. The user will fill the form fields
2. When all fields are correct, she can click the submit button to create that CA
3. The CA is created and the page is reloaded
4. If there was an error creating the CA, the user gets back to the creating page and the error is notified as yellow post-it like notice, just above the main central form. Otherwise the user navigates to the CA edit page.




Creating a new Certificate
==========================

When clicking on the main page link to create a new Cert, (+), the Cert creation form appears. Its layout is something like this:

	New Certificate for Certification Authority CA2...

					[Possible notice]

			Name:
			(Details...)
   			Duration in Days: (365)

   			(SUBMIT BUTTON)

The Address, Org. Unit and Organization details are copied from the signing CA (in this case CA2) and can only be viewed and edited by clicking on Details. Apart from that it works just like the CA creation form:

1. The user will fill the form fields
2. When all fields are correct, she can click the submit button to create that Cert
3. The Certificate is created and the page is reloaded
4. If there was an error creating the Certificate, the user gets back to the creating page and the error is notified as yellow post-it like notice, just above the main central form. Otherwise the user navigates to the Certificate edit page.


Editing a CA or Certificate
===========================

When selecting a CA or Certificate on the main page, or when they've just been created, the user navigates to the certificate editing page. It looks something like this:

	Edit Certificate server4.example.com from CA2...
	(or "Edit Certificate Authority CA2...)

	(<-Back to main page)

					[Possible notice]

			Name 	Expiration Date
			Street, Postal Code, Locality, Province, Country
			Org. Unit	Organization
   			Duration in Days

   			(DOWNLOAD KEY) (DOWNLOAD CERT) (RENEW) (CLONE) [(DELETE)]

1. The Certificate details are presented to the user (this is NOT a form)
2. Below them there are 5 links for each of the operations allowed on the Certificate or CA
3. There is also a (<- Back to main page) link to get back to the main page

> Any error will be displayed as a notice

* (DOWNLOAD KEY) downloads the Certificate's private key in PEM format
* (DOWNLOAD CERT) downloads the Certificate in PEM format
* (RENEW) reloads the edit page after renewing the certificate, changing the expiration date
* (CLONE) loads the create CA/Certificate page with currents certificate naming data
* (DELETE) after user confirmation, removes the Certificate and loads the main page

> The (DELETE) button is only shown in Certificates and empty CAs, those that currently have NO signed certificates


Settings
========

Main page's (Settings) link loads the account and notifications form:

	Email Notification Settings...

	(<-Back to main page)

					[Possible notice]

			User Name:
			Display Name:
			Email:
			Password:
			Password confirm:

			Sender Email:
			Email Password:
			Email Server URL:
			Days before expiration notice:
			Other email recipients:
			[X] Auto-renew

   			(SUBMIT BUTTON)

1. Once the form is correctly filled, the user can submit it
2. The account & email settings are saved and re-applied

> Any error will be displayed as a notice

The user's email and the additional recipients get a notification a configurable number of days before the its expiration date. The Certificate can even get renewed at the same time the email notification is sent, if the user selected auto-renew.



