// Script de inicialización de MongoDB para el servidor de administración
// Este script se ejecuta automáticamente cuando se crea el contenedor de MongoDB

// Crear base de datos
db = db.getSiblingDB('opo');

// Crear colección user
db.createCollection('user');

// Crear colección topics_uuid_map
db.createCollection('topics_uuid_map');

// Crear colección apps
db.createCollection('apps');

// Insertar datos iniciales de apps
db.apps.insertMany([
  {
    "_id": "1",
    "id": "1",
    "name": "Policía Nacional",
    "description": "Área policía nacional",
    "createdAt": new Date().toISOString(),
    "updatedAt": new Date().toISOString()
  },
  {
    "_id": "2", 
    "id": "2",
    "name": "Policía Local/Guardia Civil",
    "description": "Área policía local y guardia civil",
    "createdAt": new Date().toISOString(),
    "updatedAt": new Date().toISOString()
  }
]);

// Crear índices para optimizar consultas
db.user.createIndex({ "email": 1 }, { unique: true });
db.topics_uuid_map.createIndex({ "id": 1 }, { unique: true });
db.topics_uuid_map.createIndex({ "uuid": 1 }, { unique: true });
db.topics_uuid_map.createIndex({ "area": 1 });
db.topics_uuid_map.createIndex({ "enabled": 1 });
db.topics_uuid_map.createIndex({ "rootId": 1 });
db.topics_uuid_map.createIndex({ "area": 1, "enabled": 1 });

print('✅ Base de datos inicializada correctamente');
print('📊 Colecciones creadas: user, topics_uuid_map, apps');
print('🔍 Índices creados para optimizar consultas');
print('🏢 Apps iniciales insertadas: PN (1) y PS (2)');
