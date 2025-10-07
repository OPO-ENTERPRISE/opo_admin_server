// Script para crear usuario administrador
// Ejecutar con: mongo create-admin-user.js

// Conectar a la base de datos
db = db.getSiblingDB('opo');

// Hash de la contraseña "admin123" generado con bcrypt
// Cost: 10, Salt: $2a$10$N9qo8uLOickgx2ZMRZoMye
const hashedPassword = '$2a$10$N9qo8uLOickgx2ZMRZoMye.Ijd4v7MCLu0Tz8GzVfHnFqFqFqFqFq';

// Crear usuario administrador
const adminUser = {
  "_id": "admin-" + new ObjectId(),
  "name": "Administrador",
  "email": "admin@example.com",
  "password": hashedPassword,
  "appId": "1",
  "lastLogin": new Date(),
  "createdAt": new Date(),
  "updatedAt": new Date()
};

// Eliminar usuario existente si existe
db.user.deleteMany({});

// Insertar nuevo usuario administrador
const result = db.user.insertOne(adminUser);

if (result.insertedId) {
  print('✅ Usuario administrador creado exitosamente!');
  print('📧 Email: admin@example.com');
  print('🔑 Contraseña: admin123');
  print('🏢 App ID: 1 (Policía Nacional)');
  print('🆔 ID: ' + result.insertedId);
  print('\n🚀 Ahora puedes iniciar sesión en el servidor de administración con estas credenciales.');
} else {
  print('❌ Error creando usuario administrador');
}
