const yup = require("yup");

// Yup Schema for Regisrer Requirements
exports.registerUserScheme = yup.object().shape({
  fullname: yup.string().required("Please Enter the FullName"),
  gender: yup
    .string()
    .required("Please Enter the Gender")
    .oneOf(["male", "female"], "The Gender is not Valid"),
  age: yup.string().required("Please Enter the Age"),
  city: yup.string().required("Please Enter the City"),
});
