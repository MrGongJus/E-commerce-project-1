package controllers

import (
	"github.com/astaxie/beego"
	"github.com/gomodule/redigo/redis"
	"github.com/astaxie/beego/orm"
	"dailyFresh/models"
)

type CartController struct {
	beego.Controller
}

func AJAXRESP(this*beego.Controller,resp map[string]interface{}){
	//把数据传递到前段
	this.Data["json"] = resp

	//告诉前段以json格式接收
	this.ServeJSON()
}

func(this*CartController)HandleAddCart(){
	//获取数据
	count ,err:= this.GetInt("count")
	skuid,err2:=this.GetInt("skuid")

	resp := make(map[string]interface{})
	defer AJAXRESP(&this.Controller,resp)


	//校验数据
	if err != nil || err2 != nil{
		resp["errno"] = 1
		resp["errmsg"] = "数据传输不完整"
		return
	}
	//处理数据
	conn,err :=redis.Dial("tcp","192.168.110.81:6379")
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "redis连接失败"
		return
	}
	userName := this.GetSession("userName")
	if userName == nil{
		resp["errno"] = 3
		resp["errmsg"] = "用户未登录，请先登录"
		return
	}
	re ,err :=conn.Do("hget","cart_"+userName.(string),skuid)
	//回复助手函数
	hcount,err :=redis.Int(re,err)
	conn.Do("hset","cart_"+userName.(string),skuid,count+hcount)
	//定义一个map容器

	//给容器赋值
	resp["errno"] = 5
	resp["errmsg"] = "OK"
	//返回数据
	//this.Data["json"] = resp
	//this.ServeJSON()
}

//展示购物车页面
func(this*CartController)ShowCart(){
	//从redis中获取数据
	var goods []map[string]interface{}

	conn,err := redis.Dial("tcp","192.168.110.81:6379")
	if err != nil{
		beego.Error("redis链接失败")
		return
	}
	defer conn.Close()

	userName := this.GetSession("userName")
	re ,err := conn.Do("hkeys","cart_"+userName.(string))
	o := orm.NewOrm()

	ids ,_ :=redis.Ints(re,err)

	totalPrice := 0
	totalCount := 0
	for _,id := range ids{
		//获取购物车中商品数量
		count,_ := redis.Int(conn.Do("hget","cart_"+userName.(string),id))
		temp := make(map[string]interface{})
		temp["count"] = count

		//获取购物车中商品信息
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)
		temp["goodsSku"] = goodsSku

		littlePrice := count * goodsSku.Price
		temp["littlePrice"] = littlePrice

		totalPrice += littlePrice
		totalCount += 1

		goods = append(goods,temp)
	}

	//把数据传递给前端
	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	this.Data["goods"] = goods
	this.TplName = "cart.html"



}

//添加购物车商品数量
func(this*CartController)HandleAddCartGoods(){
	//获取数据
	skuid ,err:=this.GetInt("skuid")
	count,err2:=this.GetInt("count")

	//定义一个容器
	resp := make(map[string]interface{})
	defer AJAXRESP(&this.Controller,resp)
	//校验数据
	if err !=nil || err2 != nil{
		beego.Error("传输数据错误")
		return
	}
	//处理数据
	//向redis中添加数据
	conn,err:=redis.Dial("tcp","192.168.110.81:6379")
	if err != nil {
		beego.Error("redis链接失败")
		return
	}
	defer conn.Close()
	userName := this.GetSession("userName")
	if userName == nil{
		beego.Error("用户未登录")
		return
	}
	conn.Do("hset","cart_"+userName.(string),skuid,count)

	//给容器赋值
	resp["errno"] = 5
	resp["errmsg"] = "OK"
	////把数据传递给前段
	//this.Data["json"] = resp
	////前段接受
	//this.ServeJSON()

	//返回数据
}

//删除购物车商品
func(this*CartController)DeleteCartGoods(){
	//获取数据
	skuid,err := this.GetInt("skuid")
	//返回数据
	resp := make(map[string]interface{})
	defer AJAXRESP(&this.Controller,resp)
	//校验数据
	if err != nil{
		beego.Error("获取数据错误")
		return
	}
	//处理数据
	//从redis中删除数据
	conn,err :=redis.Dial("tcp","192.168.110.81:6379")
	if err != nil{
		beego.Error("redis链接失败")
		return
	}
	defer conn.Close()
	userName := this.GetSession("userName")
	if userName == nil{
		beego.Error("用户未登录")
		return
	}

	conn.Do("hdel","cart_"+userName.(string),skuid)


	//给map赋值
	resp["errno"] = 5
	resp["errmsg"] = "Ok"
	//把map传递给前端
	//this.Data["json"] = resp
	//this.ServeJSON()
}