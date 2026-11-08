(function () {
  // Document survives boosted body replacement; repeated asset loads must not rebind.
  if (document.researchForgeProjectFormsLoaded) return;
  document.researchForgeProjectFormsLoaded = true;

  document.addEventListener("htmx:beforeSwap", function (event) {
    const detail = event.detail;
    const config = detail && detail.requestConfig;
    const form = config && config.elt;
    if (!form || form.tagName !== "FORM" || String(config.verb).toLowerCase() !== "post") return;

    const path = form.getAttribute("hx-post");
    if (config.path !== path ||
        !["/projects/create", "/projects/open", "/projects/switch"].includes(path)) return;

    const status = document.getElementById(
      path === "/projects/switch" ? "project-switch-status" : "project-status"
    );
    if (!status || !detail.xhr) return;

    const code = detail.xhr.status;
    if (code >= 400 && code < 600) {
      status.textContent = detail.xhr.responseText;
    } else if (path === "/projects/switch" && code >= 200 && code < 400 && code !== 204) {
      status.textContent = "";
    }
  });
})();
