FROM parent:latest

COPY . /src

# Copy the linting configuration files
COPY ./build/linting /src
