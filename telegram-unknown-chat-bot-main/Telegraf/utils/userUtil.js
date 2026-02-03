const { default: axios } = require("axios");

exports.findUser = async (id) => {
  try {
    const user = await axios.get(
      `http://localhost:3000/admin/users/find/${id}`
    );

    return user.data.user;
  } catch (err) {
    return false;
  }
};

exports.isUserRegistered = async (id) => {
  try {
    const user = await axios.get(
      `http://localhost:3000/admin/users/find/${id}`
    );

    if (user.data.user) {
      return true;
    } else {
      return false;
    }
  } catch (error) {
    return false;
  }
};

exports.createUser = async (fullname, gender, age, city, telegramId) => {
  try {
    const result = await axios.post(
      "http://localhost:3000/admin/users/create",
      {
        fullname,
        gender,
        age,
        city,
        telegramId,
      }
    );

    return true;
  } catch (errr) {
    return false;
  }
};
