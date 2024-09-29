# Use a base Golang image
FROM golang:1.20

# Set up working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency installation
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project directory to the container
COPY . .

RUN ls -la /app

# Build the application
RUN go build -o rotate .

# set file permission
RUN chmod +x rotate

# Start the application
CMD ["./rotate"]
