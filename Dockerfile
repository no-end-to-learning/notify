FROM node:16-alpine

WORKDIR /app

ENV NODE_ENV production

EXPOSE 8000

COPY package.json package-lock.json ./

RUN npm install

COPY . .

CMD ["node", "src/index.js"]
