# Use the official Golang image as a base image
FROM golang:1.17

# Set the working directory inside the container
WORKDIR /app

# Copy the Go application files to the working directory
COPY . .

# Print the contents of the current directory for debugging
RUN ls -al

# Build the Go application
RUN go build -o main .

# Expose the port on which the application will run
EXPOSE 8080

# Print the contents of the /app directory for debugging
RUN ls -al /app

# Command to run the executable
CMD ["./main"]
