package sdk


// For adding categories and such
// It's stored in database as a separate entity and is always kept in app cache for fast loading
// maybe not needed? just add another field to Field to define a static group entity - needs function that then keeps that data in cache
func (e *Entity) AddStaticGroupEntity(name string, groupEntity *Entity, staticValues Data) {

}