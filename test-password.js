const bcrypt = require('bcrypt');

// Hash de la contraseña en la base de datos
const hashFromDB = "$2a$10$87IBCvasc9.KmC.zNDrWMOykMjnJHnS6ynJFNQqN53ZsJU9485/7G";

// Contraseñas a probar
const passwordsToTest = [
    "12345678",
    "admin123",
    "password",
    "superadmin",
    "123456",
    "admin",
    "test"
];

console.log("🔍 Probando contraseñas contra el hash de la BD:");
console.log("Hash: " + hashFromDB);
console.log("");

passwordsToTest.forEach(password => {
    const isMatch = bcrypt.compareSync(password, hashFromDB);
    console.log(`Contraseña: "${password}" -> ${isMatch ? '✅ CORRECTA' : '❌ Incorrecta'}`);
});

console.log("");
console.log("🔧 Generando hash para '12345678':");
const newHash = bcrypt.hashSync("12345678", 10);
console.log("Nuevo hash: " + newHash);
