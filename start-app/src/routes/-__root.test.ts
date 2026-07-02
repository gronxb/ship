import { describe, expect, it } from "vitest"

import { dashboardHead } from "./__root"

describe("root document metadata", () => {
  it("colors iOS browser safe areas with dashboard metadata", () => {
    const head = dashboardHead()

    expect(head.meta).toEqual(
      expect.arrayContaining([
        { name: "theme-color", content: "#101312" },
        { name: "apple-mobile-web-app-capable", content: "yes" },
        {
          name: "apple-mobile-web-app-status-bar-style",
          content: "black-translucent",
        },
      ])
    )
  })
})
