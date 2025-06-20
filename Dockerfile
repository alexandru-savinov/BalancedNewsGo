# Use official Node.js 20 image
FROM node:20

WORKDIR /app

# Copy only package files first for caching
COPY package.json package-lock.json* ./
RUN npm ci || npm install

# Copy the rest of the project
COPY . .

# Install Go (required for Playwright webServer)
RUN apt-get update && \
    apt-get install -y wget && \
    wget https://go.dev/dl/go1.22.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz && \
    rm go1.22.4.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:$PATH"

# Install Playwright browsers
RUN npx playwright install --with-deps

CMD ["npx", "playwright", "test"]
