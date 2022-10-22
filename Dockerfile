FROM node:16-alpine

ENV APP_ROOT /app
ENV NODE_ENV production

EXPOSE 8000

WORKDIR ${APP_ROOT}

COPY package.json package-lock.json ./

RUN npm install

COPY . .

CMD ["node", "src/index.js"]
