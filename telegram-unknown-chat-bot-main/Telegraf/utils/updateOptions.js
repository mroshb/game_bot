exports.updateOptions = (commands) => {
  if (commands == []) return { reply_markup: {} };

  // Create the keyboard markup
  const keyboard = commands.map((command) => [command]);

  // Create the message options with the keyboard
  const options = {
    reply_markup: {
      keyboard: keyboard,
      one_time_keyboard: false,
    },
  };

  return options;
};

exports.homeOptions = () => {
  const commands = ["Start searching for someone to chat"];

  if (commands == []) return { reply_markup: {} };

  // Create the keyboard markup
  const keyboard = commands.map((command) => [command]);

  // Create the message options with the keyboard
  const options = {
    reply_markup: {
      keyboard: keyboard,
      one_time_keyboard: false,
    },
  };

  return options;
};
