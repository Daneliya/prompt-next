import { readFileSync, writeFileSync } from 'node:fs'

const config = readFileSync(new URL('./config.yml', import.meta.url), 'utf8')
const version = config.match(/^  version: "(\d+\.\d+\.\d+)"/m)?.[1]
if (!version) throw new Error('build/config.yml must specify a three-part application version')

const [input, output] = process.argv.slice(2)
if (input && output) {
  const template = readFileSync(input, 'utf8')
  writeFileSync(output, template.replaceAll('{{APP_VERSION}}', version))
} else {
  process.stdout.write(version)
}
