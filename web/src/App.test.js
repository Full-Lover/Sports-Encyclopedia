import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import App from "./App.vue";

describe("App", () => {
  it("renders the map experience foundation", () => {
    const wrapper = mount(App);

    expect(wrapper.get("h1").text()).toBe("Explore teams by place.");
    expect(wrapper.text()).toContain("The interactive map foundation is ready.");
  });
});
