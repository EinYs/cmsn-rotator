# Use a base Golang image
FROM golang:1.20

# Set up working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency installation
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project directory to the container
COPY . .

# Build the rotate binary for efficiency
RUN go build -o rotate ./rotate

# Install cron
RUN apt-get update && apt-get install -y cron

# Copy the crontab file to the cron directory
COPY rotate_crontab /etc/cron.d/rotate_crontab

# Copy the script that will run the rotate commands
COPY rotate_script.sh /rotate_script.sh
RUN chmod +x /rotate_script.sh

# Give execution rights on the cron job file
RUN chmod 0644 /etc/cron.d/rotate_crontab

# Apply the cron job
RUN crontab /etc/cron.d/rotate_crontab

# Start the cron service and keep the container running
CMD cron && tail -f /dev/null
