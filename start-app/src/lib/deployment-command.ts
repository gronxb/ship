import { execFile, spawn } from "node:child_process"
import { promisify } from "node:util"

export const execFileAsync = promisify(execFile)

export async function execFileWithInput(
  command: string,
  args: readonly string[],
  input: string
): Promise<{ readonly stdout: string; readonly stderr: string }> {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args)
    let stdout = ""
    let stderr = ""

    child.stdout.setEncoding("utf8")
    child.stderr.setEncoding("utf8")
    child.stdout.on("data", (chunk: string) => {
      stdout += chunk
    })
    child.stderr.on("data", (chunk: string) => {
      stderr += chunk
    })
    child.on("error", reject)
    child.on("close", (code) => {
      if (code === 0) {
        resolve({ stdout, stderr })
        return
      }
      const error = new Error(`${command} exited with status ${code}`)
      Object.assign(error, { code, stdout, stderr })
      reject(error)
    })

    child.stdin.end(input)
  })
}
