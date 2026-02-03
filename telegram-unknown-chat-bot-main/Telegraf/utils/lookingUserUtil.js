const { default: axios } = require("axios");

exports.findLookingUser = async (id) => {
  try {
    const lookingUser = await axios.get(
      `http://localhost:3000/admin/lookingusers/find/${id}`
    );

    return lookingUser.data.lookingUser;
  } catch (err) {
    return false;
  }
};

exports.findLookingUserByGender = async (gender, id) => {
  try {
    const lookingUser = await axios.post(
      "http://localhost:3000/admin/lookingusers/findbygender/",
      { gender, telegramId: id }
    );

    return lookingUser.data.lookingUser;
  } catch (err) {
    return false;
  }
};

exports.createLookingUser = async (
  telegramId,
  fullname,
  gender,
  requestedGender
) => {
  try {
    const result = await axios.post(
      "http://localhost:3000/admin/lookingusers/create",
      {
        telegramId,
        fullname,
        gender,
        requestedGender,
      }
    );

    return result;
  } catch (err) {
    return false;
  }
};

exports.deleteLookingUser = async (id) => {
  try {
    const result = await axios.delete(
      `http://localhost:3000/admin/lookingusers/delete/${id}`
    );

    return true;
  } catch (err) {
    return false;
  }
};
