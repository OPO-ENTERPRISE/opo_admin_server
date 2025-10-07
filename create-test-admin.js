// Script para crear usuario administrador de prueba
// Este script crea un usuario con contraseña "admin123" hasheada

const bcrypt = require('bcrypt');
const { MongoClient } = require('mongodb');

async function createTestAdmin() {
    const uri = 'mongodb://localhost:27017';
    const client = new MongoClient(uri);
    
    try {
        await client.connect();
        console.log('Conectado a MongoDB');
        
        const db = client.db('opo');
        const collection = db.collection('user');
        
        // Eliminar usuarios existentes
        await collection.deleteMany({});
        console.log('Usuarios existentes eliminados');
        
        // Hash de la contraseña "admin123"
        const hashedPassword = await bcrypt.hash('admin123', 10);
        
        // Crear usuario administrador
        const adminUser = {
            _id: 'admin-test-' + Date.now(),
            name: 'Administrador',
            email: 'admin@example.com',
            password: hashedPassword,
            appId: '1',
            lastLogin: new Date(),
            createdAt: new Date(),
            updatedAt: new Date()
        };
        
        const result = await collection.insertOne(adminUser);
        console.log('✅ Usuario administrador creado exitosamente!');
        console.log('📧 Email: admin@example.com');
        console.log('🔑 Contraseña: admin123');
        console.log('🏢 App ID: 1 (Policía Nacional)');
        console.log('🆔 ID:', result.insertedId);
        
    } catch (error) {
        console.error('❌ Error:', error);
    } finally {
        await client.close();
        console.log('Conexión cerrada');
    }
}

createTestAdmin();
