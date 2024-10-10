document.addEventListener("DOMContentLoaded", function () {
  const signUpButton = document.getElementById("signUp");
  const loginButton = document.getElementById("logIn");

  //signUpButton redirect action
  signUpButton.addEventListener("click", function () {
    window.location.href = "/user/signup";
  });

  //loginButton Redirect Action
  loginButton.addEventListener("click", function () {
    window.location.href = "/user/login";
  });
});
