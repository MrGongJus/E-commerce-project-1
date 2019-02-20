package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"dailyFresh/models"
	"math"
	"github.com/gomodule/redigo/redis"
)

type GoodsController struct {
	beego.Controller
}

func(this*GoodsController)ShowIndex(){
	userName := this.GetSession("userName")
	if userName == nil{
		this.Data["userName"] = ""
	}else{
		this.Data["userName"] = userName.(string)
	}
	o := orm.NewOrm()
	//获取所有商品类型
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes

	//获取轮播图
	var lunboImage []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&lunboImage)
	this.Data["lunboImage"] = lunboImage

	//获取促销商品
	var cuxiaoGoods []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&cuxiaoGoods)
	this.Data["cuxiaoGoods"] = cuxiaoGoods

	//首页展示商品
	var goods []map[string]interface{}
	//把所有类型放进切片中
	for _,value := range goodsTypes{
		//定义一个行容器
		temp := make(map[string]interface{})
		temp["goodsType"] = value

		goods = append(goods,temp)
	}
	//把所有首页展示商品放到大容器中  goods[0]  goods[1]
	for _,value := range goods{
		//查询首页展示商品
		var textGoods []models.IndexTypeGoodsBanner
		var imageGoods []models.IndexTypeGoodsBanner
		qs := o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType","GoodsSKU").Filter("GoodsType",value["goodsType"])
		qs.Filter("DisplayType",0).All(&textGoods)
		qs.Filter("DisplayType",1).All(&imageGoods)

		value["textGoods"] = textGoods
		value["imageGoods"] = imageGoods
	}
	this.Data["goods"] = goods


	//指定视图
	this.TplName = "index.html"
}

//展示商品详情
func(this*GoodsController)ShowDetail(){
	//获取数据
	id,err :=this.GetInt("id")
	//校验数据
	if err != nil{
		beego.Error("商品不存在")
		this.Redirect("/",302)
		return
	}
	//处理数据
	o := orm.NewOrm()
	////获取操作对象
	var goods models.GoodsSKU
	////给操作对象赋值
	//goods.Id = id
	////查询
	//o.Read(&goods)
	o.QueryTable("GoodsSKU").RelatedSel("Goods","GoodsType").Filter("Id",id).One(&goods)
	//返回数据

	//获取当前商品同一类型的新品数据
	var newGoods []models.GoodsSKU
	qs :=o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",goods.GoodsType.Id)
	qs.OrderBy("Time").Limit(2,0).All(&newGoods)
	this.Data["newGoods"] = newGoods

	//登录的状态下存储用户浏览记录    redis中的list
	userName := this.GetSession("userName")
	if userName != nil{
		conn,err := redis.Dial("tcp","192.168.110.81:6379")
		if err != nil{
			beego.Error("redis链接错误")
			//用goto
		}
		conn.Do("lrem","history_"+userName.(string),0,id)

		conn.Do("lpush","history_"+userName.(string),id)
	}

	//获取类型数据
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes
	this.Data["goods"] = goods
	this.TplName = "detail.html"
}

//分页页码函数
func PageEditer(pageCount int,pageIndex int)[]int{
	var pages []int
	if pageCount < 5{
		for i := 0;i<pageCount ;i++{
			pages = append(pages, i+1)
		}
	}else if pageIndex <= 3{
		for i := 0;i<5 ;i++{
			pages = append(pages, i+1)
		}
	}else if pageIndex > pageCount - 3{
		for i := 0;i < 5;i++{
			pages = append(pages, pageCount - 4 + i)
		}
	}else{
		for i := -2; i<=2 ;i++{
			pages = append(pages, pageIndex + i)
		}
	}
	return pages
}

//展示商品列表页
func(this*GoodsController)ShowList(){
	//获取数据
	typeId,err :=this.GetInt("id")
	//校验数据u
	if err != nil {
		beego.Error("类型不存在")
		this.Redirect("/",302)
		return
	}
	//处理数据
	o := orm.NewOrm()
	//定义操作对象
	var goods []models.GoodsSKU
	//获取所有商品
	qsGoods :=o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId)
	//qsGoods.All(&goods)


	//获取类型
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes

	//获取新品数据
	var newGoods []models.GoodsSKU
	qs :=o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId)
	qs.OrderBy("Time").Limit(2,0).All(&newGoods)
	this.Data["newGoods"] = newGoods

	//实现分页
	//获取总页码和当前页码  queryseter    qs
	count,_ := qsGoods.Count()
	pageSize := 6
	pageCount := math.Ceil(float64(count) / float64(pageSize))

	pageIndex,err :=this.GetInt("pageIndex")
	if err != nil{
		pageIndex = 1
	}
	pages := PageEditer(int(pageCount),pageIndex)

	this.Data["pages"] = pages
	this.Data["pageIndex"] = pageIndex
	prePage := pageIndex - 1
	nextPage := pageIndex + 1

	//获取部分商品
	start := (pageIndex -1 ) * pageSize


	if prePage <1 {
		prePage = 1
	}
	if nextPage > int(pageCount){
		nextPage = int(pageCount)
	}


	this.Data["prePage"] = prePage
	this.Data["nextPage"] = nextPage

	sort := this.GetString("sort")
	if sort == ""{
		qsGoods.Limit(pageSize,start).All(&goods)
	}else if sort == "price"{
		qsGoods.Limit(pageSize,start).OrderBy("Price").All(&goods)
	}else{
		qsGoods.Limit(pageSize,start).OrderBy("Sales").All(&goods)
	}

	this.Data["sort"] = sort




	//返回数据
	this.Data["id"] = typeId
	this.Data["goods"] = goods
	this.TplName = "list.html"
}

//处理搜索业务
func(this*GoodsController)HandleSearch(){
	//获取数据
	searchGoods := this.GetString("searchGoods")
	//校验数据
	if searchGoods == ""{
		this.Redirect("/",302)
		return
	}
	//处理数据
	o := orm.NewOrm()
	//大于  小于  包含  以...开头   以...结尾   判空
	var goods []models.GoodsSKU
	o.QueryTable("GoodsSKU").Filter("Name__icontains",searchGoods).All(&goods)
	//返回数据
	this.Data["goods"] = goods
	this.TplName = "search.html"
}
