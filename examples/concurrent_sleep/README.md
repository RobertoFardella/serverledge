This examples demonstrates how to define a function through a custom
container image. In particular, we define a Python function that sleep for 5 seconds. 

The actual function code is in `function.py`. We also need to copy an Executor
implementation (see the docs) to the container. The file `executor.py` contains
an adapted version of the Executor implementation taken from the default Python
runtime image of Serverledge.

## Building the image

	$ docker build -t <IMAGETAG> .

## Using the image to create a function that you want

	$ ./create_function.sh <func_name> <IMAGETAG>
	
## Then, you can invoke a given number of function instances by passing it as a command-line argument.
	
	$ ./parallel_invocation_control.sh <function_name> <num_instances>
