FROM mcr.microsoft.com/playwright:v1.64.1-noble
WORKDIR /e2e
COPY tests/e2e/package*.json ./
RUN npm ci --ignore-scripts
COPY tests/e2e/ ./
CMD ["npm", "test"]
