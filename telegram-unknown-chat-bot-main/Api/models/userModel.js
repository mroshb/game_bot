const mongoose = require("mongoose");

const schemas = require("./secure/userValidation");

// Main Mongoose Schema for User
const userSchema = new mongoose.Schema({
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
  age: {
    type: String,
    required: true,
  },
  city: {
    type: String,
    required: true,
  },
  createdAt: {
    type: Date,
    default: Date.now,
  },
});

// Set User Validation in Statics
userSchema.statics.userValidation = function (body) {
  return schemas.registerUserScheme.validate(body, { abortEarly: false });
};

const User = mongoose.model("User", userSchema);

module.exports = User;
