FROM node:17-alpine

WORKDIR /

COPY executor.js /
COPY worker.js /
RUN mkdir -p /app

CMD node /executor.js



