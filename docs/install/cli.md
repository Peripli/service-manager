# Installing the smctl

## Download and add to PATH

Download the latest release from [HERE][2]. Add the `smctl` executable to your PATH.

## Example usage of CLI

```sh
# We need to connect and authenticate with a running Service Manager instance before doing anythign else
smctl login -a http://service-manager-url.com -u {user} -p {pass}

# List all brokers
smctl list-brokers
ID                                    Name  URL                             Description                                      Created               Updated
------------------------------------  ----  ------------------------------  -----------------------------------------------  --------------------  --------------------


# Registering a broker
smctl register-broker sample-broker-1 https://demobroker.domain.com/ "Service broker providing some valuable services" -b {user}:{pass}
ID                                    Name             URL                             Description                                      Created               Updated
------------------------------------  ---------------  ------------------------------  -----------------------------------------------  --------------------  --------------------
a52be735-30e5-4849-af23-83d65d592464  sample-broker-1  https://demobroker.domain.com/  Service broker providing some valuable services  2018-06-22T13:04:19Z  2018-06-22T13:04:19Z


# Registering another broker
smctl register-broker sample-broker-2 https://demobroker.domain.com/ "Another broker providing valuable services" -b {user}:{pass}
ID                                    Name             URL                             Description                                      Created               Updated
------------------------------------  ---------------  ------------------------------  -----------------------------------------------  --------------------  --------------------
a52be735-30e5-4849-af23-83d65d592464  sample-broker-1  https://demobroker.domain.com/   Service broker providing some valuable services  2018-06-22T13:04:19Z  2018-06-22T13:04:19Z
b419b538-b938-4293-86e0-7c92b0200d8e  sample-broker-2  https://demobroker.domain.com/   Another broker providing valuable services       2018-06-22T13:05:41Z  2018-06-22T13:05:41Z

```

For a list of all available commands run: ``smctl help``

## Documentation

Documentation of the Service Manager CLI and all of it's commands can be found [HERE][1].

[1]: https://github.com/Peripli/service-manager-cli/
[2]: https://github.com/Peripli/service-manager-cli/releases
