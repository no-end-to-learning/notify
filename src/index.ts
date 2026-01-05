import config from 'config'
import app from './app.js'
import { logger } from './lib/logger.js'

interface ServerConfig {
  host: string
  port: number
  baseURL: string
}

const serverConfig = config.get<ServerConfig>('server')

app.listen(serverConfig.port, serverConfig.host, () => {
  logger.info('Server listening at %s', serverConfig.baseURL)
})
