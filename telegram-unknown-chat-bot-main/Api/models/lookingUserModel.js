const mongoose = require("mongoose");

const schemas = require("./secure/lookingUserValidation");

// Main Mongoose Schema for Looking User
const lookingUserSchema = new mongoose.Schema({
  telegramId: {
    type: String,
    require: true,
    unique: true,
  },
  fullname: {
    type: String,
    required: true,
    trim: true,
  },
  gender: {
    type: String,
    require: true,
    enum: ["male", "female"],
  },
  requestedGender: {
    type: String,
    require: true,
    enum: ["male", "female"],
  },
});

// Set Looking User Validation in Statics
lookingUserSchema.statics.lookingUserValidation = function (body) {
  return schemas.lookingUserScheme.validate(body, { abortEarly: false });
};

const LookingUser = mongoose.model("LookingUser", lookingUserSchema);

module.exports = LookingUser;
