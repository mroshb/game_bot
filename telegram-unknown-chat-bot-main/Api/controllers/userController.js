const User = require("../models/userModel");
const { createError } = require("../middlewares/errors");

// GET - /admin/users - shows all the sers
exports.getAllUsers = async (req, res, next) => {
  try {
    const users = await User.find();
    const counts = await User.find().countDocuments();

    if (users == undefined) {
      throw createError(404, "", "no user found");
    }

    res.status(200).json({ users, counts });
  } catch (err) {
    next(err);
  }
};

// POST - /admin/users/create - creates a new user
exports.createUser = async (req, res, next) => {
  try {
    let { fullname, age, city, gender, telegramId } = req.body;

    await User.userValidation({ fullname, age, city, gender });

    if (gender == "پسر" || gender == "male") {
      gender = "male";
    } else {
      gender = "female";
    }

    await User.create({ fullname, age, city, gender, telegramId });

    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};

// GET - /admin/users/find/:id - finds a single user
exports.findUser = async (req, res, next) => {
  try {
    const user = await User.findOne({ telegramId: req.params.id });

    if (!user) {
      throw createError(400, "", "no user found");
    }

    res.status(200).json({ user });
  } catch (err) {
    next(err);
  }
};

// PUT - /admin/users/edit/:id - edits a user
exports.editUser = async (req, res, next) => {
  try {
    let { fullname, age, city, gender } = req.body;

    const user = await User.findOne({ telegramId: req.params.id });

    if (!user) {
      throw createError(404, "", "no user found");
    }

    await User.userValidation({ fullname, age, city, gender });

    if (gender === "پسر") {
      gender = "male";
    } else {
      gender = "female";
    }

    user.fullname = fullname;
    user.age = age;
    user.gender = gender;

    await user.save();

    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};

// DELETE - /admin/users/delete/:id - deletes a user
exports.deleteUser = async (req, res, next) => {
  try {
    const user = await User.findOne({ telegramId: req.params.id });

    if (!user) {
      throw createError(404, "", "no user found");
    }

    await User.findOneAndRemove({ telegramId: req.params.id });
    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};
