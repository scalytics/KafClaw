(function () {
  var pref = null;
  try {
    pref = localStorage.getItem("kafclaw-theme");
  } catch (e) {}

  if (pref === "dark" || pref === "light") {
    document.documentElement.setAttribute("data-theme", pref);
  } else {
    document.documentElement.removeAttribute("data-theme");
  }
})();
