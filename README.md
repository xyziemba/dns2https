# DNS loopback to DNS-over-HTTPS proxy

dns2https provides a lightweight, local DNS server that resolves requests using
Google's [DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https)
protocol.

## Installation

Mac users via Homebrew:

```
brew install xyziemba/brew/dns2https
sudo brew services start xyziemba/brew/dns2https

# Modify Wi-Fi below to use with other network services
sudo networksetup -setdnsservers Wi-Fi 127.0.0.1

# If you want to see other network devices, use the following:
networksetup -listallnetworkservices
```
