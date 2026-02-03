const yup = require("yup");

// Yup Schema for Regisrer Requirements
exports.lookingUserScheme = yup.object().shape({
  fullname: yup.string().required("Please Enter the FullName"),
  gender: yup
    .string()
    .required("Please Enter the Gender")
    .oneOf(["male", "female"], "The Gender is not Valid"),
  requestedGender: yup
    .string()
    .required("Please Enter the Gender")
    .oneOf(["male", "female"], "The Gender is not Valid"),
});
