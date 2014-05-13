# Go in the office

It is a [martini](https://github.com/go-martini/martini) web application which allows you to see how is in your office by checking MAC addresses of connected devices in your office network.

## Features

* Allows you to get start server.

## Getting started

1. Install [Go](http://golang.org/doc/install) and set [GOLANG](http://golang.org/doc/code.html#GOPATH) path.
2. Download `martini` with `oauth2` and `session` and run the server:

	go get github.com/go-martini/martini
  go get github.com/martini-contrib/oauth2
	go run server.go

3. `nmap` `sudo apt-get install nmap` `brew update && brew install nmap`

3. Download [ngok](https://ngrok.com/download) (`localtunnel` currently doesn't work) and run:

	$ unzip /path/to/ngrok.zip
	$ ./ngrok 3000

4. Go to your forwarding url like `http://23ec754d.ngrok.com`. It allows users to sign in by using GitHub account and enter MAC addresses.
5. Now you can see which users are currently connected to the office network! :)

## Tips

* Run it on free laptop in your office.
* Fill user mobile phone's MAC addresses.

## TODO

* Add tests
* Add ability users to sing in by using GitHub accounts
* Add SQLite and save users' MAC addresses
* Show who is in the office

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request
