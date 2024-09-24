# Offloading Mechanism Example

##How to use
Open the terminal and write the following command:
```bash
./setup_offloading.sh <function-name>
```


In this example, we implement the offloading mechanism by starting an **edge node** and a **cloud node**.

We have two Python functions that are copied into the container, which are as follows:

- **`executor.py`**: This file contains an adapted version of the Executor implementation taken from the default Python runtime image of Serverledge.
- **`hello.py`**: This is a simple function that prints the parameters passed to it on standard output.

## Building the Image

```bash
sudo docker build -t roberto-image .
```

## Infrastructure Setup

```bash
sudo ./../../bin/serverledge ./confEdge.yaml
sudo ./../../bin/serverledge ./confCloud.yaml
```

## Using the Image to Create the Function Request and Its Invocation

```bash
./../../bin/serverledge-cli create -f robertoFunc --memory 256 --runtime custom \
    --custom_image roberto-image
./../../bin/serverledge-cli invoke -f robertoFunc --params_file \
./encoded_JSON_parameters/input_hello.json
```

## Logs

If there are issues with the function, you can view the container log messages using the following command:

```bash
docker container logs <Container-name>
```
---

**Notes:**
- To automate the entire process, use the bash script setup_offloading.sh
- Replace `<Container-name>` with the actual name or ID of your Docker container when accessing the logs.
- Make sure that the paths to the `serverledge` binaries and configuration files (`confEdge.yaml` and `confCloud.yaml`) are correct relative to your current working directory.

