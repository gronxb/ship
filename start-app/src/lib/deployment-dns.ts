import { Resolver } from "node:dns/promises"

export async function resolveIPv4(
  host: string,
  server: string
): Promise<readonly string[]> {
  const resolver = new Resolver()
  resolver.setServers([server])
  return resolver.resolve4(host)
}
