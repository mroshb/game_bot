const LookingUser = require("../models/lookingUserModel");
const { createError } = require("../middlewares/errors");

// POST - /admin/lookingusers/create - creates a new looking user
exports.createLookigUser = async (req, res, next) => {
  try {
    let { fullname, gender, telegramId, requestedGender } = req.body;

    await LookingUser.lookingUserValidation({
      fullname,
      requestedGender,
      gender,
    });

    if (gender == "پسر" || gender == "male") {
      gender = "male";
    } else {
      gender = "female";
    }

    if (requestedGender == "پسر" || requestedGender == "male") {
      requestedGender = "male";
    } else {
      requestedGender = "female";
    }

    await LookingUser.create({ requestedGender, gender, fullname, telegramId });

    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};

// GET - /admin/lookingusers/find/:id - finds a single looking user
exports.findLookingUser = async (req, res, next) => {
  try {
    const lookingUser = await LookingUser.findOne({
      telegramId: req.params.id,
    });

    if (!lookingUser) {
      throw createError(400, "", "no looking user found");
    }

    res.status(200).json({ lookingUser });
  } catch (err) {
    next(err);
  }
};

// DELETE - /admin/lookingusers/delete/:id - deletes a looking user
exports.deleteLookingUser = async (req, res, next) => {
  try {
    const lookingUser = await LookingUser.findOne({
      telegramId: req.params.id,
    });

    if (!lookingUser) {
      throw createError(404, "", "no looking user found");
    }

    await LookingUser.findOneAndRemove({ telegramId: req.params.id });
    res.status(200).json({ message: "done" });
  } catch (err) {
    next(err);
  }
};

// GET - /admin/lookingusers/findbygender - finds a single looking user
exports.findLookingUserByGender = async (req, res, next) => {
  try {
    if (!req.body.gender) {
      throw createError(400, "", "gender is required");
    }

    const lookingUser = await LookingUser.findOne({
      gender: req.body.gender,
    });

    if (!lookingUser) {
      throw createError(400, "", "no looking user found");
    }

    res.status(200).json({ lookingUser });
  } catch (err) {
    next(err);
  }
};
